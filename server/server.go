package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var requestTimeout = 200 * time.Millisecond
var dbTimeout = 10 * time.Millisecond

type USDBRL struct {
	Code       string `json:"code"`
	Codein     string `json:"codein"`
	Name       string `json:"name"`
	High       string `json:"high"`
	Low        string `json:"low"`
	VarBid     string `json:"varBid"`
	PctChange  string `json:"pctChange"`
	Bid        string `json:"bid"`
	Ask        string `json:"ask"`
	Timestamp  string `json:"timestamp"`
	CreateDate string `json:"create_date"`
}

type ExchangeRateIn struct {
	USDBRL USDBRL `json:"USDBRL"`
}

type ExchangeRateOut struct {
	Bid string `json:"bid"`
}

type App struct {
	Db *sql.DB
}

func main() {
	app, err := initializeApp()
	if err != nil {
		log.Fatal(err)
	}

	defer app.Db.Close()
	
	// Iniciando o servidor
	http.HandleFunc("/cotacao", app.ExchangeRateHandler)
	http.ListenAndServe(":8080", nil)
}

// Função para inicializar a aplicação
func initializeApp() (*App, error) {
    // Conectando ao banco de dados SQLite3
    db, err := sql.Open("sqlite3", "./server.db")
    if err != nil {
        return nil, err
    }

    // Criando o banco de dados e a tabela
    if err := createSchema(db); err != nil {
        return nil, err
    }

    return &App{Db: db}, nil
}

// Função para criar o esquema do banco de dados
func createSchema(db *sql.DB) error {
    schema := `
    CREATE TABLE IF NOT EXISTS ExchangeRates (
		Code       VARCHAR(10) NOT NULL,
		Codein     VARCHAR(10) NOT NULL,
		Name       VARCHAR(50) NOT NULL,
		High       VARCHAR(20) NOT NULL,
		Low        VARCHAR(20) NOT NULL,
		VarBid     VARCHAR(20) NOT NULL,
		PctChange  VARCHAR(20) NOT NULL,
		Bid        VARCHAR(20) NOT NULL,
		Ask        VARCHAR(20) NOT NULL,
		Timestamp  INTEGER NOT NULL,
		CreateDate DATE NOT NULL DEFAULT CURRENT_DATE
	);`
    _, err := db.Exec(schema)
    return err
}

func (app *App) ExchangeRateHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	select {
	case <-ctx.Done():
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	default:
		exchangeRate, err := ExchangeRate(ctx)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Inserindo os dados na tabela
		data := exchangeRate.USDBRL
		log.Println(data)
		ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
		defer cancel()
		_, err = insertData(ctx, app.Db, data)
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		exchangeRateOut := ExchangeRateOut{Bid: data.Bid}
		json.NewEncoder(w).Encode(exchangeRateOut)
	}
}

func insertData(ctx context.Context, db *sql.DB, data USDBRL) (int64, error) {

	insertSql := "INSERT INTO ExchangeRates (Code, Codein, Name, High, Low, VarBid, PctChange, Bid, Ask, Timestamp, CreateDate) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"

	row, err := db.ExecContext(ctx, insertSql, data.Code, data.Codein, data.Name, data.High, data.Low, data.VarBid, data.PctChange, data.Bid, data.Ask, data.Timestamp, data.CreateDate)
	if err != nil {
		return 0, err
	}

	return row.LastInsertId()
}

func ExchangeRate(ctx context.Context) (*ExchangeRateIn, error) {
	url := "https://economia.awesomeapi.com.br/json/last/USD-BRL"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)

	if err != nil {
		log.Println(err)
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer resp.Body.Close()
	res, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var data ExchangeRateIn
	err = json.Unmarshal(res, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

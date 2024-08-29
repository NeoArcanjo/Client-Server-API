package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

var requestTimeout = 300 * time.Millisecond

type ExchangeRate struct {
	Bid        string `json:"bid"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/cotacao", nil)
	if err != nil {
		panic(err)
	}	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao ler resposta: %v\n", err)
	}
	var data ExchangeRate
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao decodificar resposta: %v\n", err)
	}
	file, err := os.Create("cotacao.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao criar arquivo: %v\n", err)
	}
	defer file.Close()
	_, err = file.WriteString(fmt.Sprintf("Dólar: R$ %s\n", data.Bid))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao escrever no arquivo: %v\n", err)
	} else {
		fmt.Println("Arquivo criado com sucesso!")
	}
}
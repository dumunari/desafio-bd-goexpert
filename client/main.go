package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type Cotacao struct {
	USDBRL `json:"USDBRL"`
}

type USDBRL struct {
	Bid string `json:"bid"`
}

func main() {
	cotacao, err := getCotacao()
	if err != nil {
		panic(err)
	}

	err = saveToFile(cotacao)
	if err != nil {
		panic(err)
	}
}

func getCotacao() (*Cotacao, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	resultCh := make(chan *Cotacao, 1)
	errCh := make(chan error, 1)

	go func() {
		req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/cotacao", nil)
		if err != nil {
			errCh <- err
			return
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			errCh <- err
			return
		}
		defer res.Body.Close()

		var cotacao Cotacao
		if res.StatusCode != http.StatusOK {
			errCh <- err
			return
		}

		body, err := io.ReadAll(res.Body)
		if err != nil {
			errCh <- err
			return
		}

		err = json.Unmarshal([]byte(body), &cotacao)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- &cotacao
	}()
	select {
	case cotacao := <-resultCh:
		return cotacao, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		log.Println(ctx.Err().Error())
		return nil, ctx.Err()
	}
}

func saveToFile(cotacao *Cotacao) error {
	file, err := os.Create("cotacao.txt")
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write([]byte(fmt.Sprintf("Dólar: %s", cotacao.USDBRL.Bid)))
	if err != nil {
		return err
	}

	return nil
}

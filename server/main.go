package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Cotacao struct {
	ID     uint `gorm:"primaryKey" json:"-"`
	USDBRL `gorm:"embedded" json:"USDBRL"`
}

type USDBRL struct {
	Code       string `json:"code"`
	CodeIn     string `json:"codein"`
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

func (Cotacao) TableName() string {
	return "cotacoes"
}

func main() {
	cleanup()
	http.HandleFunc("/cotacao", getCotacao)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Error starting server: %s\n", err.Error())
	}
}

func getCotacao(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	cotacao, err := retrieveCotacao()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Failed to retrieve data: ` + err.Error() + `"}`))
		return
	}

	err = saveCotacao(cotacao)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Failed to save data: ` + err.Error() + `"}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(cotacao)
	if err != nil {
		println(err)
	}
}

func retrieveCotacao() (*Cotacao, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	resultCh := make(chan *Cotacao, 1)
	errCh := make(chan error, 1)

	go func() {
		req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
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

		body, err := io.ReadAll(res.Body)
		if err != nil {
			errCh <- err
			return
		}

		var cotacao Cotacao
		err = json.Unmarshal(body, &cotacao)
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
		log.Printf("Error retrieving from api: %s\n", ctx.Err().Error())
		return nil, ctx.Err()
	}
}

func saveCotacao(cotacao *Cotacao) error {
	db, err := bootstrapDb()
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	go func() {
		err := db.WithContext(ctx).Create(cotacao).Error
		if err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Printf("Error saving to db: %s\n", ctx.Err().Error())
		return ctx.Err()
	}

}

func cleanup() {
	db, err := bootstrapDb()
	if err != nil {
		panic(err)
	}
	db.Migrator().DropTable("cotacoes")
}

func bootstrapDb() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("cotacoes.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&Cotacao{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	_ "github.com/lib/pq"
)

type Bias struct {
	ID            int    `json:"id"`
	Image         string `json:"engTitle"`
	Paid          bool   `json:"paid"`
	Category      string `json:"category"`
	WikipediaLink string `json:"wikipediaLink"`
}

func parseBiasesJSONFiles(files []string) ([]Bias, error) {
	var allBiases []Bias

	for _, file := range files {

		fileData, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("не удалось прочитать файл %s: %v", file, err)
		}

		var biases []Bias
		err = json.Unmarshal(fileData, &biases)
		if err != nil {
			return nil, fmt.Errorf("не удалось распарсить JSON из файла %s: %v", file, err)
		}

		allBiases = append(allBiases, biases...)
	}

	return allBiases, nil
}

func getBiasByID(db *sql.DB, biasID int) (*Bias, error) {
	var bias Bias
	query := `SELECT id, image, paid, category, wikipedia_link FROM biases WHERE id = $1`
	err := db.QueryRow(query, biasID).Scan(&bias.ID, &bias.Image, &bias.Paid, &bias.Category, &bias.WikipediaLink)
	if err != nil {
		return nil, err
	}
	return &bias, nil
}

func updateBias(w http.ResponseWriter, r *http.Request, db *sql.DB) {

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "ID не указан", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Неверный ID", http.StatusBadRequest)
		return
	}

	var bias Bias
	err = json.NewDecoder(r.Body).Decode(&bias)
	if err != nil {
		http.Error(w, "Ошибка при декодировании JSON", http.StatusBadRequest)
		return
	}

	query := `UPDATE biases SET image = $1, paid = $2, category = $3, wikipedia_link = $4 WHERE id = $5`
	_, err = db.Exec(query, bias.Image, bias.Paid, bias.Category, bias.WikipediaLink, id)
	if err != nil {
		http.Error(w, "Ошибка при обновлении искажения", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Искажение с ID %d обновлено", id)
}

func deleteBias(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Получаем ID из URL
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "ID не указан", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Неверный ID", http.StatusBadRequest)
		return
	}

	// Выполняем SQL-запрос на удаление
	query := `DELETE FROM biases WHERE id = $1`
	_, err = db.Exec(query, id)
	if err != nil {
		http.Error(w, "Ошибка при удалении искажения", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Искажение с ID %d удалено", id)
}

// Функция пагинации
func paginateBiases(biases []Bias, page, pageSize int) ([]Bias, error) {
	if page <= 0 || pageSize <= 0 {
		return nil, fmt.Errorf("пагинация: неверные параметры страницы или размера страницы")
	}

	start := (page - 1) * pageSize
	if start >= len(biases) {
		return nil, nil
	}

	end := start + pageSize
	if end > len(biases) {
		end = len(biases)
	}

	return biases[start:end], nil
}

// Получение списка искажений
func getBiases(w http.ResponseWriter, r *http.Request, db *sql.DB, biases []Bias) {
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("pageSize")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize <= 0 {
		pageSize = 10
	}

	paginatedBiases, err := paginateBiases(biases, page, pageSize)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка при пагинации: %v", err), http.StatusInternalServerError)
		return
	}

	// Отправка результата
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(paginatedBiases)
}

// Запуск сервера
func main() {
	// Подключение к базе данных
	connStr := "host=localhost port=5432 user=Igor52 password=1331 dbname=ccbb sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	// Загружаем данные из JSON файлов
	files := []string{"biases-rus.json", "biases-eng.json"}
	biases, err := parseBiasesJSONFiles(files)
	if err != nil {
		log.Fatalf("Ошибка при работе с JSON-файлами: %v", err)
	}

	// Выводим распарсенные данные для теста
	fmt.Println("Данные из обоих JSON файлов загружены.")
	for _, bias := range biases {
		fmt.Printf("ID: %d, Category: %s\n", bias.ID, bias.Category)
	}

	// Создание HTTP-обработчиков
	http.HandleFunc("/biases", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			getBiases(w, r, db, biases) // Получение искажений с пагинацией
		} else if r.Method == "PUT" {
			updateBias(w, r, db) // Обновление искажения
		} else if r.Method == "DELETE" {
			deleteBias(w, r, db) // Удаление искажения
		} else {
			http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		}
	})

	// Запуск сервера
	fmt.Println("Запуск сервера на порту 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

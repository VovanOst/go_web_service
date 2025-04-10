package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// Тестовая структура для ответа сервера
type TestCase struct {
	Query      string
	OrderField string
	OrderBy    int
	Limit      int
	Offset     int
	WantStatus int
	WantError  string
}

func TestSearchServer(t *testing.T) {
	// Запускаем тестовый сервер
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	// Набор тестов
	cases := []TestCase{
		// 1. Базовый тест (без фильтров)
		{Query: "", OrderField: "Name", OrderBy: OrderByAsc, Limit: 5, Offset: 0, WantStatus: http.StatusOK},

		// 2. Фильтр по Query (ищем конкретное имя)
		{Query: "John", OrderField: "Name", OrderBy: OrderByAsc, Limit: 5, Offset: 0, WantStatus: http.StatusOK},

		// 3. Ошибка: неверный order_field
		{Query: "", OrderField: "UnknownField", OrderBy: OrderByAsc, Limit: 5, Offset: 0, WantStatus: http.StatusBadRequest, WantError: "ErrorBadOrderField"},

		// 4. Ошибка: отрицательный offset
		{Query: "", OrderField: "Name", OrderBy: OrderByAsc, Limit: 5, Offset: -1, WantStatus: http.StatusBadRequest},

		// 5. Ошибка: слишком большой offset (пустой результат)
		{Query: "", OrderField: "Name", OrderBy: OrderByAsc, Limit: 5, Offset: 1000, WantStatus: http.StatusOK},

		// 6. Проверяем сортировку по возрасту (Age)
		{Query: "", OrderField: "Age", OrderBy: OrderByAsc, Limit: 5, Offset: 0, WantStatus: http.StatusOK},

		// 7. Проверяем сортировку по ID (Id)
		{Query: "", OrderField: "Id", OrderBy: OrderByDesc, Limit: 5, Offset: 0, WantStatus: http.StatusOK},
	}

	// Запускаем тесты
	for i, tc := range cases {
		t.Run("Test case "+strconv.Itoa(i), func(t *testing.T) {
			url := ts.URL + "/?query=" + tc.Query +
				"&order_field=" + tc.OrderField +
				"&order_by=" + strconv.Itoa(tc.OrderBy) +
				"&limit=" + strconv.Itoa(tc.Limit) +
				"&offset=" + strconv.Itoa(tc.Offset)

			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("Ошибка при запросе: %v", err)
			}
			defer resp.Body.Close()

			// Проверяем статус-код
			if resp.StatusCode != tc.WantStatus {
				t.Errorf("Ожидался статус %d, но получили %d", tc.WantStatus, resp.StatusCode)
			}

			// Проверяем JSON-ошибку
			if tc.WantStatus == http.StatusBadRequest {
				body, _ := ioutil.ReadAll(resp.Body)
				var errResp struct {
					Error string `json:"error"`
				}
				_ = json.Unmarshal(body, &errResp)
				if errResp.Error != tc.WantError {
					t.Errorf("Ожидалась ошибка %s, но получили %s", tc.WantError, errResp.Error)
				}
			}
		})
	}
}

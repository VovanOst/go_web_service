package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	//"net/http/httptest"
	//"testing"
	"time"
)

// Создаём тестовый сервер
func createTestServer(handlerFunc http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handlerFunc)
}

// Тесты для FindUsers
func TestFindUsers(t *testing.T) {
	tests := []struct {
		name         string
		serverFunc   http.HandlerFunc
		request      SearchRequest
		wantErr      string
		wantUsers    []User
		wantNextPage bool
	}{
		// ✅ 1. Успешный запрос
		{
			name: "Success Request",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				users := []User{
					{Id: 1, FirstName: "John", LastName: "Doe", Age: 30, About: "A guy"},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(users)
			},
			request: SearchRequest{
				Limit:  5,
				Offset: 0,
			},
			wantUsers: []User{
				{Id: 1, FirstName: "John", LastName: "Doe", Age: 30, About: "A guy"},
			},
			wantNextPage: false,
		},

		// ❌ 2. Ошибка: Bad AccessToken (401)
		{
			name: "Unauthorized (401)",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			request: SearchRequest{
				Limit:  5,
				Offset: 0,
			},
			wantErr: "Bad AccessToken",
		},

		// ❌ 3. Ошибка: Internal Server Error (500)
		{
			name: "Internal Server Error (500)",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			request: SearchRequest{
				Limit:  5,
				Offset: 0,
			},
			wantErr: "SearchServer fatal error",
		},

		// ❌ 4. Ошибка: невалидный JSON от сервера
		{
			name: "Invalid JSON Response",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("{invalid json}"))
			},
			request: SearchRequest{
				Limit:  5,
				Offset: 0,
			},
			wantErr: "cant unpack result json: invalid character 'i' looking for beginning of object key string",
		},

		// ❌ 5. Ошибка: таймаут сервера
		{
			name: "Timeout Error",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(time.Second * 2) // эмулируем задержку
			},
			request: SearchRequest{
				Limit:  5,
				Offset: 0,
			},
			wantErr: "timeout for limit=6&offset=0&order_by=0&order_field=&query=",
		},

		// ❌ 6. Ошибка: Bad Request (400) с неверным полем сортировки
		{
			name: "Bad Request - Invalid OrderField",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				resp := SearchErrorResponse{Error: "ErrorBadOrderField"}
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(resp)
			},
			request: SearchRequest{
				Limit:      5,
				Offset:     0,
				OrderField: "invalid_field",
			},
			wantErr: "OrderFeld invalid_field invalid",
		},

		// ❌ 7. Ошибка: Bad Request (400) с неизвестной ошибкой
		{
			name: "Bad Request - Unknown Error",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				resp := SearchErrorResponse{Error: "UnknownError"}
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(resp)
			},
			request: SearchRequest{
				Limit:  5,
				Offset: 0,
			},
			wantErr: "unknown bad request error: UnknownError",
		},

		// ❌ 8. Ошибка: отрицательный limit
		{
			name: "Negative Limit",
			request: SearchRequest{
				Limit:  -1,
				Offset: 0,
			},
			wantErr: "limit must be > 0",
		},

		// ❌ 9. Ошибка: отрицательный offset
		{
			name: "Negative Offset",
			request: SearchRequest{
				Limit:  5,
				Offset: -1,
			},
			wantErr: "offset must be > 0",
		},
		// ✅ 10. Успешный запрос c limit more 25
		{
			name: "limit more 25",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				users := []User{
					{Id: 1, FirstName: "John", LastName: "Doe", Age: 30, About: "A guy"},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(users)
			},
			request: SearchRequest{
				Limit:  100,
				Offset: 0,
			},
			wantUsers: []User{
				{Id: 1, FirstName: "John", LastName: "Doe", Age: 30, About: "A guy"},
			},
			wantNextPage: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Если сервер нужен — запускаем его
			var srvURL string
			if tc.serverFunc != nil {
				ts := createTestServer(tc.serverFunc)
				defer ts.Close()
				srvURL = ts.URL
			}

			client := &SearchClient{URL: srvURL}
			result, err := client.FindUsers(tc.request)

			// Проверяем ошибки
			if err != nil {
				if tc.wantErr == "" {
					t.Errorf("Unexpected error: %v", err)
				} else if err.Error() != tc.wantErr && !errors.Is(err, fmt.Errorf(tc.wantErr)) {
					t.Errorf("Expected error '%s', got '%s'", tc.wantErr, err.Error())
				}
				return
			} else if tc.wantErr != "" {
				t.Errorf("Expected error '%s', but got none", tc.wantErr)
			}

			// Проверяем данные
			if result != nil {
				if len(result.Users) != len(tc.wantUsers) {
					t.Errorf("Expected %d users, got %d", len(tc.wantUsers), len(result.Users))
				}
				if result.NextPage != tc.wantNextPage {
					t.Errorf("Expected NextPage: %v, got %v", tc.wantNextPage, result.NextPage)
				}
			}
		})
	}
}

type fakeTransport struct{}

func (f *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, errors.New("some unknown error occurred")
}

func TestFindUsers_UnknownError(t *testing.T) {
	// Создаём клиент с кастомным транспортом, который всегда выдаёт ошибку
	clientMock := &http.Client{
		Transport: &fakeTransport{},
	}

	// Подменяем оригинальный клиент
	oldClient := client
	client = clientMock
	defer func() { client = oldClient }() // Восстанавливаем после теста

	searchClient := &SearchClient{
		URL: "http://example.com",
	}

	_, err := searchClient.FindUsers(SearchRequest{Limit: 5, Offset: 0})
	if err == nil || !strings.Contains(err.Error(), "some unknown error occurred") { //!strings.Contains(err.Error(), "cant unpack error json")
		t.Errorf("Expected 'unknown error some unknown error occurred', got: %v", err)
	}
}

func TestFindUsers_CantUnpackErrorJSON(t *testing.T) {
	// Фейковый сервер, который возвращает невалидный JSON при 400 Bad Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{invalid json}`)) // Невалидный JSON
	}))
	defer server.Close()

	searchClient := &SearchClient{
		URL: server.URL,
	}

	_, err := searchClient.FindUsers(SearchRequest{Limit: 5, Offset: 0})
	if err == nil || err.Error() != "cant unpack error json: invalid character 'i' looking for beginning of object key string" {
		t.Errorf("Expected JSON parse error, got: %v", err)
	}
}

func TestFindUsers_NextPageTrue(t *testing.T) {
	// Создаём фейковый сервер, который возвращает `req.Limit` пользователей
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		limit := query.Get("limit")

		// Проверяем, что сервер получил `limit=4`
		if limit != "4" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		users := []User{
			{Id: 1, FirstName: "Alice", LastName: "Smith", Age: 25},
			{Id: 2, FirstName: "Bob", LastName: "Johnson", Age: 30},
			{Id: 3, FirstName: "Charlie", LastName: "Brown", Age: 35},
			{Id: 4, FirstName: "David", LastName: "White", Age: 40}, // 4-й пользователь
		}

		json.NewEncoder(w).Encode(users) // Возвращаем 4 пользователя (Limit+1)
	}))
	defer server.Close()

	searchClient := &SearchClient{
		URL: server.URL,
	}

	req := SearchRequest{Limit: 3, Offset: 0} // Ожидаемый лимит 3 -> сервер вернёт 4
	resp, err := searchClient.FindUsers(req)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Проверяем, что `NextPage` стал `true`
	if !resp.NextPage {
		t.Errorf("expected NextPage to be true, got false")
	}

	// Проверяем, что в `Users` осталось 3 пользователя (4-й удалён)
	if len(resp.Users) != 3 {
		t.Errorf("expected 3 users, got %d", len(resp.Users))
	}
}

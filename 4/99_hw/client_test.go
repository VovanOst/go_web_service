package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

func CheckServer(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("id")

	// Симуляция таймаута
	if key == "timeout" {
		time.Sleep(2 * time.Second) // Дольше таймаута клиента
		http.Error(w, "timeout", http.StatusGatewayTimeout)
		return
	}

	switch key {
	case "1":
		// Симуляция 500 ошибки сервера
		w.WriteHeader(http.StatusInternalServerError)
		return
	case "100500":
		// Симуляция 400 Bad Request с JSON-ошибкой
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"status": 400, "err": "bad_balance"}`)
		return
	case "__unauthorized":
		// Симуляция сломанного JSON
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"status": 401`) // Некорректный JSON
		return
	case "__internal_error":
		// Еще один случай 500 ошибки
		w.WriteHeader(http.StatusInternalServerError)
		return
	default:
		// По умолчанию 200 OK с пустым телом
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"status": 200, "balance": 100500}`)
	}
}
func SearchServer(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	orderField := r.URL.Query().Get("order_field")
	orderBy, _ := strconv.Atoi(r.URL.Query().Get("order_by"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	//fmt.Println("%v&[%v]&[%v]&[%v]___c", query, orderField, orderBy, limit, offset)
	if orderField == "" {
		orderField = "Name"
	}
	if offset < 0 {
		http.Error(w, fmt.Sprintf("offset less 0: %s", orderField), http.StatusBadRequest)
	}

	ts := httptest.NewServer(http.HandlerFunc(CheckServer))
	if orderField != "id" && orderField != "Age" && orderField != "Name" {
		http.Error(w, fmt.Sprintf("unknown order_field: %s", orderField), http.StatusBadRequest)
		//t.Errorf("[%d] unexpected error: %#v", caseNum, err)
	}

	if offset < 0 {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"status": 200, "balance": 100500}`)
	}

	data, err := ioutil.ReadFile("dataset.xml")
	if err != nil {
		http.Error(w, "cannot read dataset", http.StatusInternalServerError)
		return
	}

	var users Users
	if err := xml.Unmarshal(data, &users); err != nil {
		http.Error(w, "cannot parse dataset", http.StatusInternalServerError)
		return
	}
	var result []User
	for _, user := range users.List {

		name := user.FirstName + " " + user.LastName

		if query == "" || strings.Contains(name, query) || strings.Contains(user.About, query) {
			result = append(result, User{
				Id:        user.Id,
				FirstName: user.FirstName,
				LastName:  user.LastName,
				Age:       user.Age,
				About:     user.About,
			})
			//fmt.Printf("Имя %s запрос %s ->прошло %v\n", name, query, strings.Contains(name, query))
		}
	}

	sort.Slice(result, func(i, j int) bool {
		switch orderField {
		case "Id":
			if orderBy == OrderByDesc {
				return result[i].Id > result[j].Id
			}
			return result[i].Id < result[j].Id
		case "Age":
			if orderBy == OrderByDesc {
				return result[i].Age > result[j].Age
			}
			return result[i].Age < result[j].Age
		case "Name":
			nameI := result[i].FirstName + " " + result[i].LastName
			nameJ := result[j].FirstName + " " + result[j].LastName
			if orderBy == OrderByDesc {
				return nameI > nameJ
			}
			return nameI < nameJ
		}
		return false
	})

	/*if offset > len(result) {
		offset = len(result)
	}*/
	if offset > len(result) {
		offset = 0
	}
	if limit+offset > len(result) {
		limit = len(result) - offset
	}

	response := result[offset : offset+limit]
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	ts.Close()
}

func (c *Cart) Checkout(id string) (*CheckoutResult, error) {
	url := c.PaymentApiURL + "?id=" + id
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	result := &CheckoutResult{}

	err = json.Unmarshal(data, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func normalizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ") // Убирает лишние пробелы
}

func TestRequest(t *testing.T) {
	cases := []TestCase{
		{ID: "1",
			SearchRequest: SearchRequest{
				Limit:      100,
				Offset:     0,
				Query:      "Hilda",
				OrderField: "Name",
				OrderBy:    1,
			},
			Result: &SearchResponse{
				Users: []User{
					{Id: 1,
						FirstName: "Hilda",
						LastName:  "Mayer",
						Age:       21,
						About:     "Sit commodo consectetur minim amet ex. Elit aute mollit fugiat labore sint ipsum dolor cupidatat qui reprehenderit. Eu nisi in exercitation culpa sint aliqua nulla nulla proident eu. Nisi reprehenderit anim cupidatat dolor incididunt laboris mollit magna commodo ex. Cupidatat sit id aliqua amet nisi et voluptate voluptate commodo ex eiusmod et nulla velit.\n"},
				},
				NextPage: false,
			},
			IsError:    false,
			StatusCode: http.StatusInternalServerError,
		},
		{ID: "2",
			SearchRequest: SearchRequest{
				Limit:      100,
				Offset:     -1,
				Query:      "Hilda",
				OrderField: "Name",
				OrderBy:    1,
			},
			Result:  nil,
			IsError: false,
		},
		{ID: "3",
			SearchRequest: SearchRequest{
				Limit:      -1,
				Offset:     0,
				Query:      "Hilda",
				OrderField: "Name",
				OrderBy:    1,
			},
			Result:  nil,
			IsError: false,
		},
		{ID: "__unauthorized",
			SearchRequest: SearchRequest{
				Limit:      1,
				Offset:     0,
				Query:      "Hilda",
				OrderField: "Name",
				OrderBy:    1,
			},
			Result:     nil,
			IsError:    true,
			StatusCode: http.StatusUnauthorized,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	for caseNum, item := range cases {
		c := &SearchClient{
			AccessToken: "test_token",
			URL:         ts.URL,
		}
		result, err := c.FindUsers(item.SearchRequest)
		//expect1 := item.Result.Users[0].About
		//expect2 := result.Users[0].About

		/*if expect1 != expect2 {
			t.Errorf("[%d] wrong result, expected %q, got %q", caseNum, expect1, expect2)
		}*/

		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}

		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("[%d] wrong result, expected %#v, got %#v", caseNum, item.Result, result)
		}
	}
}

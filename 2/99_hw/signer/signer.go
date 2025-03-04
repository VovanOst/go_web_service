package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Определяем тип job – каждая функция в конвейере имеет вид:
// func(in, out chan interface{})
//type job func(in, out chan interface{})

// ExecutePipeline соединяет набор функций (воркеров) в цепочку.
// Каждый следующий job получает на вход канал, в который предыдущий записал результаты.
func ExecutePipeline(jobs ...job) {
	in := make(chan interface{})
	// Для каждого job создаём канал для передачи результатов и запускаем горутину:
	for _, j := range jobs {
		out := make(chan interface{})
		go func(j job, in, out chan interface{}) {
			defer close(out)
			j(in, out)
		}(j, in, out)
		in = out
	}
	// Дожидаемся завершения финального этапа – просто опустошаем последний канал.
	for range in {
	}
}

// Глобальный мьютекс для синхронизации вызова DataSignerMd5 (не может работать параллельно)
var muMd5 sync.Mutex

// SingleHash вычисляет crc32(data)+"~"+crc32(md5(data)) для каждого входного значения.
// Для каждого значения запускется своя горутина.
func SingleHash(in, out chan interface{}) {
	var wg sync.WaitGroup
	for v := range in {
		wg.Add(1)
		// Каждый входной элемент обрабатываем в отдельной горутине:
		go func(v interface{}) {
			defer wg.Done()
			data := fmt.Sprintf("%v", v)

			// Вычисляем md5(data) строго по очереди
			muMd5.Lock()
			md5Data := DataSignerMd5(data)
			muMd5.Unlock()

			// Параллельно вычисляем crc32(data) и crc32(md5(data))
			crc32Ch := make(chan string)
			crc32md5Ch := make(chan string)

			go func() {
				crc32Ch <- DataSignerCrc32(data)
			}()
			go func() {
				crc32md5Ch <- DataSignerCrc32(md5Data)
			}()

			crc32Data := <-crc32Ch
			crc32md5Data := <-crc32md5Ch

			result := crc32Data + "~" + crc32md5Data
			out <- result
		}(v)
	}
	wg.Wait()
}

// MultiHash для каждого входного значения (строка, полученная из SingleHash)
// параллельно вычисляет 6 значений: crc32(strconv.Itoa(i)+data) для i от 0 до 5,
// а затем конкатенирует их в одну строку в порядке от 0 до 5.
func MultiHash(in, out chan interface{}) {
	var wg sync.WaitGroup
	for v := range in {
		wg.Add(1)
		go func(v interface{}) {
			defer wg.Done()
			data := fmt.Sprintf("%v", v)
			var results [6]string
			var innerWg sync.WaitGroup
			for i := 0; i < 6; i++ {
				innerWg.Add(1)
				// Чтобы не было проблемы с замыканием, передаём i в параметрах функции.
				go func(i int) {
					defer innerWg.Done()
					// Вычисляем crc32(strconv.Itoa(i)+data)
					results[i] = DataSignerCrc32(strconv.Itoa(i) + data)
				}(i)
			}
			innerWg.Wait()
			// Собираем результаты в одну строку (порядок от 0 до 5 гарантирован)
			out <- strings.Join(results[:], "")
		}(v)
	}
	wg.Wait()
}

// CombineResults собирает все входные данные, сортирует их в лексикографическом порядке,
// а затем объединяет с помощью символа "_".
func CombineResults(in, out chan interface{}) {
	var results []string
	for v := range in {
		results = append(results, fmt.Sprintf("%v", v))
	}
	sort.Strings(results)
	out <- strings.Join(results, "_")
}

// Функции DataSignerMd5 и DataSignerCrc32 предоставляются извне (например, в common.go).
//var DataSignerMd5 func(data string) string
//var DataSignerCrc32 func(data string) string

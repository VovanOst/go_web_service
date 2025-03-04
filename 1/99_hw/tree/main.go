package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// dirTree – точка входа для обхода дерева каталогов.
// Параметры:
//
//	out – куда выводить результат,
//	path – корневой путь для обхода,
//	printFiles – печатать ли файлы (true) или только каталоги.
func dirTree(out io.Writer, path string, printFiles bool) error {
	return walkDir(out, path, "", printFiles)
}

// walkDir рекурсивно обходит каталог path, выводя дерево с префиксом prefix.
// Если printFiles == false, то выводятся только каталоги.
func walkDir(out io.Writer, path string, prefix string, printFiles bool) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	// Фильтруем элементы: если не нужно печатать файлы – оставляем только каталоги.
	var filtered []os.DirEntry
	for _, entry := range entries {
		// Игнорируем .DS_Store (на MacOS)
		if entry.Name() == ".DS_Store" {
			continue
		}
		if !printFiles && !entry.IsDir() {
			continue
		}
		filtered = append(filtered, entry)
	}

	// Сортировка по имени
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Name() < filtered[j].Name()
	})

	// Обходим отфильтрованные элементы
	for i, entry := range filtered {
		connector := "├───"
		// Если последний элемент, то меняем символ ветвления
		if i == len(filtered)-1 {
			connector = "└───"
		}

		if entry.IsDir() {
			// Вывод имени директории
			fmt.Fprintf(out, "%s%s%s\n", prefix, connector, entry.Name())
			// Обновляем префикс для следующего уровня
			newPrefix := prefix
			if i == len(filtered)-1 {
				newPrefix += "\t"
			} else {
				newPrefix += "│\t"
			}
			// Рекурсивный вызов для дочерней директории
			if err := walkDir(out, filepath.Join(path, entry.Name()), newPrefix, printFiles); err != nil {
				return err
			}
		} else {
			// Файл: получаем информацию о файле
			info, err := entry.Info()
			if err != nil {
				return err
			}
			sizeStr := ""
			if info.Size() == 0 {
				sizeStr = " (empty)"
			} else {
				sizeStr = fmt.Sprintf(" (%db)", info.Size())
			}
			// Вывод имени файла с размером
			fmt.Fprintf(out, "%s%s%s%s\n", prefix, connector, entry.Name(), sizeStr)
		}
	}

	return nil
}

func main() {
	// Парсим аргументы командной строки.
	// Ожидается: go run main.go <path> [-f]
	args := os.Args[1:]
	if len(args) < 1 || len(args) > 2 {
		fmt.Fprintf(os.Stderr, "Usage: go run main.go <path> [-f]\n")
		os.Exit(1)
	}

	path := args[0]
	printFiles := false
	if len(args) == 2 && args[1] == "-f" {
		printFiles = true
	}

	// Запускаем обход дерева.
	if err := dirTree(os.Stdout, path, printFiles); err != nil {
		panic(err)
	}
}

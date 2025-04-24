package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"hw11_shopql/graph/model"
)

type Cart struct {
	Items map[string]*model.CartItem
	Order []string
}

// Service хранит все данные приложения в памяти.
type Service struct {
	Catalogs map[string]*model.Catalog
	Items    map[string]*model.Item
	Sellers  map[string]*model.Seller
	Carts    map[string]*Cart
}

// raw структуры для первичного парсинга вложенного JSON каталога и товаров
// ID и seller_id приходят как json.Number

type rawCatalogNode struct {
	ID     json.Number      `json:"id"`
	Name   string           `json:"name"`
	Childs []rawCatalogNode `json:"childs,omitempty"`
	Items  []rawItemNode    `json:"items,omitempty"`
}

type rawItemNode struct {
	ID       json.Number `json:"id"`
	Name     string      `json:"name"`
	Stock    int         `json:"in_stock"`
	SellerID json.Number `json:"seller_id"`
}

type rawSeller struct {
	ID    json.Number `json:"id"`
	Name  string      `json:"name"`
	Deals int         `json:"deals"`
}

type rawData struct {
	Catalog rawCatalogNode `json:"catalog"`
	Sellers []rawSeller    `json:"sellers"`
}

// NewService читает testdata.json, парсит вложенную структуру каталога и создает модели.
func NewService() *Service {
	log.Println("starting server")
	s := &Service{
		Catalogs: make(map[string]*model.Catalog),
		Items:    make(map[string]*model.Item),
		Sellers:  make(map[string]*model.Seller),
		Carts:    make(map[string]*Cart),
	}

	// Открываем и декодируем JSON
	path := filepath.Join("testdata.json")
	f, err := os.Open(path)
	if err != nil {
		panic(fmt.Errorf("cannot open %s: %w", path, err))
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	dec.UseNumber()
	var raw rawData
	if err := dec.Decode(&raw); err != nil && err != io.EOF {
		panic(fmt.Errorf("cannot decode %s: %w", path, err))
	}

	// 1) Сначала создаем всех продавцов
	for _, rs := range raw.Sellers {
		sid, _ := rs.ID.Int64()
		seller := &model.Seller{
			ID:    int(sid),
			Name:  rs.Name,
			Deals: rs.Deals,
			Items: []*model.Item{},
		}
		s.Sellers[rs.ID.String()] = seller
	}

	// 2) Рекурсивно обходим каталог и товары
	var traverse func(node rawCatalogNode, parent *model.Catalog)
	traverse = func(node rawCatalogNode, parent *model.Catalog) {
		cid, _ := node.ID.Int64()
		cat := &model.Catalog{
			ID:     int(cid),
			Name:   node.Name,
			Parent: parent,
			Childs: []*model.Catalog{},
			Items:  []*model.Item{},
		}
		s.Catalogs[node.ID.String()] = cat
		if parent != nil {
			parent.Childs = append(parent.Childs, cat)
		}

		// товары в разделе
		for _, ri := range node.Items {
			// Преобразуем json.Number в строку — ключ для карты s.Items и s.Sellers
			iidStr := ri.ID.String()
			// Преобразуем json.Number в int — для поля model.Item.ID
			iidInt64, _ := ri.ID.Int64()
			iid := int(iidInt64)

			// Аналогично для seller_id
			sidStr := ri.SellerID.String()
			seller, ok := s.Sellers[sidStr]
			if !ok {
				// Здесь можно panic или логировать ошибку — но если данных согласованы, ok==true
				panic(fmt.Errorf("seller with id %s not found", sidStr))
			}

			// Создаём модель товара
			item := &model.Item{
				ID:          iid,
				Name:        ri.Name,
				Parent:      cat,
				Seller:      seller,
				Stock:       ri.Stock,
				InStockText: "", // будет вычисляться в резолвере
			}

			// Кладём в карту по строковому ключу
			s.Items[iidStr] = item
			// Добавляем в список каталога
			cat.Items = append(cat.Items, item)
			// Добавляем в список товаров продавца
			seller.Items = append(seller.Items, item)
		}

		// рекурсивно обходим дочерние разделы
		for _, child := range node.Childs {
			traverse(child, cat)
		}
	}

	traverse(raw.Catalog, nil)
	return s
}

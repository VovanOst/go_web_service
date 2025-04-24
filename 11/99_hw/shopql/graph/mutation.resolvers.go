package graph

import (
	"hw11_shopql/graph/model"
	"hw11_shopql/service"
)

func cartItemsSlice(c *service.Cart) []*model.CartItem {
	var out []*model.CartItem
	for _, key := range c.Order {
		if ci, ok := c.Items[key]; ok {
			out = append(out, ci)
		}
	}
	return out
}

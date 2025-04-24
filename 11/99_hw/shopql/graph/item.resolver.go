package graph

import (
	"context"
	"errors"
	"hw11_shopql/graph/model"
)

/*type itemResolver struct{ *Resolver }*/

/*func (r *Resolver) Item() ItemResolver { return &itemResolver{r} }*/

/*func (r *itemResolver) InCart(ctx context.Context, obj *model.Item) (int, error) {
	userID, err := GetUserIDFromContext(ctx)
	if err != nil {
		return 0, errors.New("User not authorized")
	}
	key := strconv.Itoa(obj.ID)
	userCart, ok := r.Svc.Carts[userID]
	if !ok {
		return 0, nil
	}
	if ci, exist := userCart[key]; exist {
		return ci.Quantity, nil
	}
	return 0, nil
}*/

func (r *itemResolver) Seller(ctx context.Context, obj *model.Item) (*model.Seller, error) {
	if obj.Seller == nil {
		return nil, errors.New("seller not found")
	}
	return obj.Seller, nil
}

/*func (r *itemResolver) InStockText(ctx context.Context, obj *model.Item) (string, error) {
	switch {
	case obj.Stock <= 1:
		return "мало", nil
	case obj.Stock >= 2 && obj.Stock <= 3:
		return "хватает", nil
	default:
		return "много", nil
	}
}*/

func mapCartItemsToSlice(cart map[string]*model.CartItem) []*model.CartItem {
	var cartItems []*model.CartItem
	for _, ci := range cart {
		cartItems = append(cartItems, ci)
	}
	return cartItems
}

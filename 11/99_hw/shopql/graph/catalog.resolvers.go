package graph

/*type catalogResolver struct{ *Resolver }

func (r *Resolver) Catalog() CatalogResolver { return &catalogResolver{r} }*/

/*func (r *catalogResolver) Items(
	ctx context.Context,
	obj *model.Catalog,
	limit *int,
	offset *int,
) ([]*model.Item, error) {
	log.Printf("[DEBUG] Catalog.Items for catalog %d, limit=%v, offset=%v", obj.ID, limit, offset)

	all := obj.Items

	// default offset = 0
	off := 0
	if offset != nil {
		off = *offset
	}
	// default limit = 3
	lim := 3
	if limit != nil {
		lim = *limit
	}

	end := off + lim
	if end > len(all) {
		end = len(all)
	}

	sliced := all[off:end]
	log.Printf("[DEBUG] Returning %d of %d items: ids=%v", len(sliced), len(all), func() []int {
		ids := make([]int, len(sliced))
		for i, it := range sliced {
			ids[i] = it.ID
		}
		return ids
	}())

	return sliced, nil
}*/

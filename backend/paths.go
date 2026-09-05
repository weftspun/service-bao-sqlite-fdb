package backend

import (
	"context"
	"fmt"

	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
)

func paths(b *backend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "query/" + framework.GenericNameRegex("name"),
			Fields: map[string]*framework.FieldSchema{
				"name": {
					Type:        framework.TypeString,
					Description: "Registered catalog query name.",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleQueryRead,
				},
			},
			HelpSynopsis: "Run one catalog query and return its rows.",
		},
		{
			Pattern: "queries/?$",
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{
					Callback: b.handleQueriesList,
				},
			},
			HelpSynopsis: "List catalog query names.",
		},
	}
}

func (b *backend) handleQueryRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	name := data.Get("name").(string)
	entry, ok := b.catalog.Lookup(name)
	if !ok {
		return logical.ErrorResponse("query %q not registered", name), nil
	}

	args, err := bindArgs(entry, req.Data)
	if err != nil {
		return logical.ErrorResponse("bind args: %v", err), nil
	}

	rows, err := b.store.Query(ctx, entry.SQL, args...)
	if err != nil {
		return nil, fmt.Errorf("run query %q: %w", name, err)
	}
	return &logical.Response{
		Data: map[string]interface{}{
			"name": name,
			"rows": rows,
		},
	}, nil
}

func (b *backend) handleQueriesList(_ context.Context, _ *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	return logical.ListResponse(b.catalog.Names()), nil
}

func bindArgs(entry *CatalogEntry, in map[string]interface{}) ([]interface{}, error) {
	out := make([]interface{}, 0, len(entry.Args))
	for _, name := range entry.Args {
		v, ok := in[name]
		if !ok {
			return nil, fmt.Errorf("missing arg %q", name)
		}
		out = append(out, v)
	}
	return out, nil
}

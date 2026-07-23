package jsplugin

import (
	"context"
	"fmt"

	"github.com/dop251/goja"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/plugins"
)

// JSPlugin implements WidgetPlugin by running a JavaScript file via goja.
//
// The script must export a fetchData(orgID, config) function that returns a plain object.
// It may optionally export a schema() function returning an array of ConfigField objects.
// A synchronous db.query(sql, ...args) helper is available in the script.
type JSPlugin struct {
	id     string
	schema []plugins.ConfigField
	src    string
}

func New(id, src string) (*JSPlugin, error) {
	p := &JSPlugin{id: id, src: src}
	schema, err := p.loadSchema()
	if err != nil {
		return nil, err
	}
	p.schema = schema
	return p, nil
}

func (p *JSPlugin) ID() string                    { return p.id }
func (p *JSPlugin) Schema() []plugins.ConfigField { return p.schema }

func (p *JSPlugin) FetchData(ctx context.Context, db *pgxpool.Pool, orgID string, cfg map[string]any) (any, error) {
	vm := goja.New()
	if err := p.exposeDB(vm, ctx, db); err != nil {
		return nil, err
	}
	if _, err := vm.RunString(p.src); err != nil {
		return nil, fmt.Errorf("jsplugin %s: %w", p.id, err)
	}
	var fetchData goja.Callable
	if err := vm.ExportTo(vm.Get("fetchData"), &fetchData); err != nil {
		return nil, fmt.Errorf("jsplugin %s: fetchData not exported: %w", p.id, err)
	}
	val, err := fetchData(goja.Undefined(), vm.ToValue(orgID), vm.ToValue(cfg))
	if err != nil {
		return nil, fmt.Errorf("jsplugin %s: %w", p.id, err)
	}
	return val.Export(), nil
}

// loadSchema runs the script once without a DB to read the optional schema() export.
func (p *JSPlugin) loadSchema() ([]plugins.ConfigField, error) {
	vm := goja.New()
	vm.Set("db", vm.NewObject()) // stub — schema() must not query the DB
	if _, err := vm.RunString(p.src); err != nil {
		return nil, fmt.Errorf("jsplugin %s: compile: %w", p.id, err)
	}
	schemaFn := vm.Get("schema")
	if schemaFn == nil || goja.IsUndefined(schemaFn) {
		return nil, nil
	}
	var fn goja.Callable
	if err := vm.ExportTo(schemaFn, &fn); err != nil {
		return nil, nil
	}
	val, err := fn(goja.Undefined())
	if err != nil {
		return nil, fmt.Errorf("jsplugin %s: schema(): %w", p.id, err)
	}
	var fields []plugins.ConfigField
	if err := vm.ExportTo(val, &fields); err != nil {
		return nil, fmt.Errorf("jsplugin %s: schema() must return an array: %w", p.id, err)
	}
	return fields, nil
}

// exposeDB mounts a synchronous db.query(sql, ...args) function onto the VM.
func (p *JSPlugin) exposeDB(vm *goja.Runtime, ctx context.Context, db *pgxpool.Pool) error {
	dbObj := vm.NewObject()
	dbObj.Set("query", func(call goja.FunctionCall) goja.Value {
		sql := call.Argument(0).String()
		args := make([]any, len(call.Arguments)-1)
		for i, a := range call.Arguments[1:] {
			args[i] = a.Export()
		}
		rows, err := db.Query(ctx, sql, args...)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		defer rows.Close()

		var result []map[string]any
		for rows.Next() {
			vals, err := rows.Values()
			if err != nil {
				panic(vm.NewGoError(err))
			}
			row := make(map[string]any, len(rows.FieldDescriptions()))
			for i, fd := range rows.FieldDescriptions() {
				row[fd.Name] = vals[i]
			}
			result = append(result, row)
		}
		if rows.Err() != nil {
			panic(vm.NewGoError(rows.Err()))
		}
		return vm.ToValue(result)
	})
	return vm.Set("db", dbObj)
}

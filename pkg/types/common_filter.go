package types

import (
	"fmt"
	"strings"

	"gorm.io/gorm/clause"
)

type CommonFilterOperator string

const (
	CommonFilterOperatorEq        CommonFilterOperator = "eq"
	CommonFilterOperatorNotEq     CommonFilterOperator = "not_eq"
	CommonFilterOperatorLt        CommonFilterOperator = "lt"
	CommonFilterOperatorLte       CommonFilterOperator = "lte"
	CommonFilterOperatorGt        CommonFilterOperator = "gt"
	CommonFilterOperatorGte       CommonFilterOperator = "gte"
	CommonFilterOperatorDateRange CommonFilterOperator = "date_range"
	CommonFilterOperatorRange     CommonFilterOperator = "range"
	CommonFilterOperatorIn        CommonFilterOperator = "in"
)

type CommonFilter struct {
	Field    string               `json:"field"`
	Operator CommonFilterOperator `json:"operator"`
	Values   []any                `json:"values"`
	Filters  []CommonFilter       `json:"filters"`
}

// Build constructs a GORM expression.
func (f *CommonFilter) Build(builder clause.Builder) {
	if len(f.Values) == 0 {
		return
	}

	value := f.Values[0]

	switch f.Operator {
	case CommonFilterOperatorEq:
		// Handle JSON operator fields (containing -> or ->> operators)
		if strings.Contains(f.Field, "->") {
			// Use raw SQL expression for JSON operators
			clause.Expr{SQL: fmt.Sprintf("%s = ?", f.Field), Vars: []interface{}{value}}.Build(builder)
		} else {
			// Use standard equality for regular fields
			clause.Eq{Column: f.Field, Value: value}.Build(builder)
		}
	case CommonFilterOperatorNotEq:
		clause.NotConditions{Exprs: []clause.Expression{clause.Eq{Column: f.Field, Value: value}}}.Build(builder)
	case CommonFilterOperatorLt:
		clause.Lt{Column: f.Field, Value: value}.Build(builder)
	case CommonFilterOperatorLte:
		clause.Lte{Column: f.Field, Value: value}.Build(builder)
	case CommonFilterOperatorGt:
		clause.Gt{Column: f.Field, Value: value}.Build(builder)
	case CommonFilterOperatorGte:
		clause.Gte{Column: f.Field, Value: value}.Build(builder)
	case CommonFilterOperatorRange:
		if len(f.Values) < 2 {
			return
		}

		clause.And(clause.Gte{Column: f.Field, Value: f.Values[0]}, clause.Lte{Column: f.Field, Value: f.Values[1]}).Build(builder)
	case CommonFilterOperatorIn:
		clause.IN{Column: f.Field, Values: f.Values}.Build(builder)
	default:
		return
	}
}

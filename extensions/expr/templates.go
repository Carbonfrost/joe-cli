package expr

import (
	"fmt"
	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/internal/synopsis"
)

type exprData struct {
	Name        string
	Synopsis    *synopsisWrapper[*synopsis.Expr]
	HelpText    string
	ManualText  string
	Description string
	Data        map[string]interface{}
}

type exprDataCategory struct {
	Undocumented bool
	Category     string
	VisibleExprs []*exprData
}

type exprDescriptionData struct {
	VisibleExprs    []*exprData
	ExprsByCategory []*exprDataCategory
}

type synopsisWrapper[T synopsis.Stringer] struct {
	s T
}

func wrapSynopsis[T synopsis.Stringer](s T) *synopsisWrapper[T] {
	return &synopsisWrapper[T]{s}
}

func (s *synopsisWrapper[T]) String() string {
	buf := cli.NewBuffer()
	s.s.WriteTo(buf)
	return buf.String()
}

func exprAdapter(val *Expr) *exprData {
	syn := val.newSynopsis()
	return &exprData{
		Name:        val.Name,
		HelpText:    renderHelp(syn.Usage),
		Description: fmt.Sprint(val.Description),
		ManualText:  val.ManualText,
		Synopsis:    wrapSynopsis(syn),
		Data:        val.Data,
	}
}

func exprDescription(e *Expression) *exprDescriptionData {
	exprs := e.VisibleExprs()
	var (
		visibleExprs = func(items []*Expr) []*exprData {
			res := make([]*exprData, 0, len(items))
			for _, a := range items {
				res = append(res, exprAdapter(a))
			}
			return res
		}
		visibleExprCategories = func(items exprsByCategory) []*exprDataCategory {
			res := make([]*exprDataCategory, 0, len(items))
			for _, a := range items {
				res = append(res, &exprDataCategory{
					Category:     a.Category,
					Undocumented: a.Undocumented(),
					VisibleExprs: visibleExprs(a.VisibleExprs()),
				})
			}
			if len(res) == 1 && res[0].Category == "" {
				return nil
			}
			return res
		}
	)
	return &exprDescriptionData{
		VisibleExprs:    visibleExprs(exprs),
		ExprsByCategory: visibleExprCategories(groupExprsByCategory(exprs)),
	}
}

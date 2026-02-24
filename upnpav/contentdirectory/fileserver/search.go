// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

package fileserver

import (
	"strings"

	"github.com/ethulhu/helix/upnpav"
	"github.com/ethulhu/helix/upnpav/contentdirectory/search"
)

func matches(obj interface{}, criteria search.Criteria) bool {
	switch c := criteria.(type) {
	case search.Everything:
		return true
	case search.Query:
		return exprMatches(obj, c.Expr)
	default:
		return false
	}
}

func exprMatches(obj interface{}, expr search.Expr) bool {
	switch e := expr.(type) {
	case search.LogicExpr:
		return logicExprMatches(obj, e)
	case search.BinaryExpr:
		return binaryExprMatches(obj, e)
	case search.ExistsExpr:
		return existsExprMatches(obj, e)
	}
	return false
}

func logicExprMatches(obj interface{}, expr search.LogicExpr) bool {
	switch expr.Op {
	case search.And:
		for _, sub := range expr.SubExprs {
			if !exprMatches(obj, sub) {
				return false
			}
		}
		return true
	case search.Or:
		for _, sub := range expr.SubExprs {
			if exprMatches(obj, sub) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func binaryExprMatches(obj interface{}, expr search.BinaryExpr) bool {
	if expr.Property == "upnp:class" {
		switch item := obj.(type) {
		case upnpav.Item:
			if expr.Op == search.DerivedFrom {
				return classIsDerivedFrom(item.Class, upnpav.Class(expr.Operand))
			}
		}
	}

	var propValue string

	switch item := obj.(type) {
	case upnpav.Item:
		if expr.Property == "dc:title" {
			propValue = item.Title
		}
	case upnpav.Container:
		if expr.Property == "dc:title" {
			propValue = item.Title
		}
	}

	switch expr.Op {
	case search.Equal:
		return propValue == expr.Operand
	case search.NotEqual:
		return propValue != expr.Operand
	case search.Contains:
		return strings.Contains(propValue, expr.Operand)
	case search.DoesNotContain:
		return !strings.Contains(propValue, expr.Operand)
	default:
		return false
	}
}

func existsExprMatches(obj interface{}, expr search.ExistsExpr) bool {
	var hasProp bool

	switch item := obj.(type) {
	case upnpav.Item:
		if expr.Property == "dc:title" {
			hasProp = item.Title != ""
		}
	case upnpav.Container:
		if expr.Property == "dc:title" {
			hasProp = item.Title != ""
		}
	}

	return hasProp == expr.Exists
}

func classIsDerivedFrom(class, ancestor upnpav.Class) bool {
	return class == ancestor || strings.HasPrefix(string(class), string(ancestor)+".")
}

// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package search

import (
	"strings"

	"github.com/ethulhu/helix/upnpav"
)

// ContainsFunc reports whether the value of property contains operand. It
// decides the `contains` operator, and its negation decides `doesNotContain`.
// Property is always given in its canonical form, so the aliases `id`,
// `parentID`, and `refID` arrive as `@id`, `@parentID`, and `@refID`.
type ContainsFunc func(property, value, operand string) bool

// ContainsFold is the default ContainsFunc: a case-insensitive substring test.
func ContainsFold(_, value, operand string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(operand))
}

// Matches returns true if the given UPnP object (Item or Container) matches the search criteria.
func Matches(obj interface{}, criteria Criteria) bool {
	return MatchesWith(obj, criteria, ContainsFold)
}

// MatchesWith is like Matches, but decides the `contains` and `doesNotContain`
// operators with contains, so a server can match titles the way its own
// search does. A nil contains falls back to ContainsFold.
func MatchesWith(obj interface{}, criteria Criteria, contains ContainsFunc) bool {
	if criteria == nil {
		return true
	}
	if contains == nil {
		contains = ContainsFold
	}
	switch c := criteria.(type) {
	case Everything:
		return true
	case Query:
		return exprMatches(obj, c.Expr, contains)
	default:
		return false
	}
}

func exprMatches(obj interface{}, expr Expr, contains ContainsFunc) bool {
	switch e := expr.(type) {
	case LogicExpr:
		return logicExprMatches(obj, e, contains)
	case BinaryExpr:
		return binaryExprMatches(obj, e, contains)
	case ExistsExpr:
		return existsExprMatches(obj, e)
	}
	return false
}

func logicExprMatches(obj interface{}, expr LogicExpr, contains ContainsFunc) bool {
	switch expr.Op {
	case And:
		for _, sub := range expr.SubExprs {
			if !exprMatches(obj, sub, contains) {
				return false
			}
		}
		return true
	case Or:
		for _, sub := range expr.SubExprs {
			if exprMatches(obj, sub, contains) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func binaryExprMatches(obj interface{}, expr BinaryExpr, contains ContainsFunc) bool {
	var class upnpav.Class
	var title string
	var id string
	var parentID string
	var refID string

	switch o := obj.(type) {
	case upnpav.Item:
		class = o.Class
		title = o.Title
		id = string(o.ID)
		parentID = string(o.Parent)
		refID = o.RefID
	case *upnpav.Item:
		class = o.Class
		title = o.Title
		id = string(o.ID)
		parentID = string(o.Parent)
		refID = o.RefID
	case upnpav.Container:
		class = o.Class
		title = o.Title
		id = string(o.ID)
		parentID = string(o.Parent)
	case *upnpav.Container:
		class = o.Class
		title = o.Title
		id = string(o.ID)
		parentID = string(o.Parent)
	default:
		return false
	}

	if expr.Property == "upnp:class" {
		targetClass := upnpav.Class(expr.Operand)
		switch expr.Op {
		case DerivedFrom:
			return ClassIsDerivedFrom(class, targetClass)
		case Equal:
			return class == targetClass
		case NotEqual:
			return class != targetClass
		}
	}

	var property, propValue string
	switch expr.Property {
	case "dc:title":
		property, propValue = "dc:title", title
	case "@id", "id":
		property, propValue = "@id", id
	case "@parentID", "parentID":
		property, propValue = "@parentID", parentID
	case "@refID", "refID":
		property, propValue = "@refID", refID
	default:
		return false
	}

	switch expr.Op {
	case Equal:
		return strings.EqualFold(propValue, expr.Operand) || propValue == expr.Operand
	case NotEqual:
		return !strings.EqualFold(propValue, expr.Operand) && propValue != expr.Operand
	case Contains:
		return contains(property, propValue, expr.Operand)
	case DoesNotContain:
		return !contains(property, propValue, expr.Operand)
	default:
		return false
	}
}

func existsExprMatches(obj interface{}, expr ExistsExpr) bool {
	var hasProp bool

	var title string
	var refID string
	var id string

	switch o := obj.(type) {
	case upnpav.Item:
		title = o.Title
		refID = o.RefID
		id = string(o.ID)
	case *upnpav.Item:
		title = o.Title
		refID = o.RefID
		id = string(o.ID)
	case upnpav.Container:
		title = o.Title
		id = string(o.ID)
	case *upnpav.Container:
		title = o.Title
		id = string(o.ID)
	default:
		return false
	}

	switch expr.Property {
	case "dc:title":
		hasProp = title != ""
	case "@refID", "refID":
		hasProp = refID != ""
	case "@id", "id":
		hasProp = id != ""
	default:
		hasProp = false
	}

	return hasProp == expr.Exists
}

func ClassIsDerivedFrom(class, ancestor upnpav.Class) bool {
	return class == ancestor || strings.HasPrefix(string(class), string(ancestor)+".")
}

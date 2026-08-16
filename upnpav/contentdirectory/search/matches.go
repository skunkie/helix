// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package search

import (
	"strings"

	"github.com/ethulhu/helix/upnpav"
)

// Matches returns true if the given UPnP object (Item or Container) matches the search criteria.
func Matches(obj interface{}, criteria Criteria) bool {
	if criteria == nil {
		return true
	}
	switch c := criteria.(type) {
	case Everything:
		return true
	case Query:
		return exprMatches(obj, c.Expr)
	default:
		return false
	}
}

func exprMatches(obj interface{}, expr Expr) bool {
	switch e := expr.(type) {
	case LogicExpr:
		return logicExprMatches(obj, e)
	case BinaryExpr:
		return binaryExprMatches(obj, e)
	case ExistsExpr:
		return existsExprMatches(obj, e)
	}
	return false
}

func logicExprMatches(obj interface{}, expr LogicExpr) bool {
	switch expr.Op {
	case And:
		for _, sub := range expr.SubExprs {
			if !exprMatches(obj, sub) {
				return false
			}
		}
		return true
	case Or:
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

func binaryExprMatches(obj interface{}, expr BinaryExpr) bool {
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

	var propValue string
	switch expr.Property {
	case "dc:title":
		propValue = title
	case "@id", "id":
		propValue = id
	case "@parentID", "parentID":
		propValue = parentID
	case "@refID", "refID":
		propValue = refID
	default:
		return false
	}

	switch expr.Op {
	case Equal:
		return strings.EqualFold(propValue, expr.Operand) || propValue == expr.Operand
	case NotEqual:
		return !strings.EqualFold(propValue, expr.Operand) && propValue != expr.Operand
	case Contains:
		return strings.Contains(strings.ToLower(propValue), strings.ToLower(expr.Operand))
	case DoesNotContain:
		return !strings.Contains(strings.ToLower(propValue), strings.ToLower(expr.Operand))
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

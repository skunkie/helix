// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package search

import (
	"testing"

	"github.com/ethulhu/helix/upnpav"
)

func TestMatches(t *testing.T) {
	item := upnpav.Item{
		ID:     "item-1",
		Parent: "folder-0",
		Class:  upnpav.Class("object.item.videoItem.movie"),
		Title:  "The Matrix",
		RefID:  "item-ref-1",
	}
	container := upnpav.Container{
		ID:     "folder-0",
		Parent: "-1",
		Class:  upnpav.Class("object.container.storageFolder"),
		Title:  "Movies",
	}

	tests := []struct {
		name     string
		obj      interface{}
		criteria Criteria
		want     bool
	}{
		{
			name:     "nil criteria matches anything",
			obj:      item,
			criteria: nil,
			want:     true,
		},
		{
			name:     "Everything criteria matches anything",
			obj:      item,
			criteria: Everything{},
			want:     true,
		},
		{
			name:     "unsupported object type returns false",
			obj:      "not an item or container",
			criteria: Query{Expr: BinaryExpr{Property: "dc:title", Op: Equal, Operand: "test"}},
			want:     false,
		},
		{
			name: "title exact match on Item",
			obj:  item,
			criteria: Query{
				Expr: BinaryExpr{Property: "dc:title", Op: Equal, Operand: "The Matrix"},
			},
			want: true,
		},
		{
			name: "title case-insensitive match on Item pointer",
			obj:  &item,
			criteria: Query{
				Expr: BinaryExpr{Property: "dc:title", Op: Equal, Operand: "the matrix"},
			},
			want: true,
		},
		{
			name: "title not equal on Container",
			obj:  container,
			criteria: Query{
				Expr: BinaryExpr{Property: "dc:title", Op: NotEqual, Operand: "Series"},
			},
			want: true,
		},
		{
			name: "title contains on Container pointer",
			obj:  &container,
			criteria: Query{
				Expr: BinaryExpr{Property: "dc:title", Op: Contains, Operand: "ovi"},
			},
			want: true,
		},
		{
			name: "title does not contain on Item",
			obj:  item,
			criteria: Query{
				Expr: BinaryExpr{Property: "dc:title", Op: DoesNotContain, Operand: "Alien"},
			},
			want: true,
		},
		{
			name: "id and @id match",
			obj:  item,
			criteria: Query{
				Expr: LogicExpr{
					Op: And,
					SubExprs: []Expr{
						BinaryExpr{Property: "id", Op: Equal, Operand: "item-1"},
						BinaryExpr{Property: "@id", Op: Equal, Operand: "item-1"},
					},
				},
			},
			want: true,
		},
		{
			name: "parentID and @parentID match",
			obj:  item,
			criteria: Query{
				Expr: LogicExpr{
					Op: And,
					SubExprs: []Expr{
						BinaryExpr{Property: "parentID", Op: Equal, Operand: "folder-0"},
						BinaryExpr{Property: "@parentID", Op: Equal, Operand: "folder-0"},
					},
				},
			},
			want: true,
		},
		{
			name: "refID and @refID match",
			obj:  item,
			criteria: Query{
				Expr: LogicExpr{
					Op: And,
					SubExprs: []Expr{
						BinaryExpr{Property: "refID", Op: Equal, Operand: "item-ref-1"},
						BinaryExpr{Property: "@refID", Op: Equal, Operand: "item-ref-1"},
					},
				},
			},
			want: true,
		},
		{
			name: "class derived from matching",
			obj:  item,
			criteria: Query{
				Expr: BinaryExpr{Property: "upnp:class", Op: DerivedFrom, Operand: "object.item.videoItem"},
			},
			want: true,
		},
		{
			name: "class exact equal match",
			obj:  container,
			criteria: Query{
				Expr: BinaryExpr{Property: "upnp:class", Op: Equal, Operand: "object.container.storageFolder"},
			},
			want: true,
		},
		{
			name: "class not equal match",
			obj:  container,
			criteria: Query{
				Expr: BinaryExpr{Property: "upnp:class", Op: NotEqual, Operand: "object.item"},
			},
			want: true,
		},
		{
			name: "logic Or expression matching",
			obj:  item,
			criteria: Query{
				Expr: LogicExpr{
					Op: Or,
					SubExprs: []Expr{
						BinaryExpr{Property: "dc:title", Op: Equal, Operand: "Non-Existent"},
						BinaryExpr{Property: "dc:title", Op: Equal, Operand: "The Matrix"},
					},
				},
			},
			want: true,
		},
		{
			name: "logic Or expression non-matching",
			obj:  item,
			criteria: Query{
				Expr: LogicExpr{
					Op: Or,
					SubExprs: []Expr{
						BinaryExpr{Property: "dc:title", Op: Equal, Operand: "Non-Existent"},
						BinaryExpr{Property: "dc:title", Op: Equal, Operand: "Another Movie"},
					},
				},
			},
			want: false,
		},
		{
			name: "exists true on title",
			obj:  item,
			criteria: Query{
				Expr: ExistsExpr{Property: "dc:title", Exists: true},
			},
			want: true,
		},
		{
			name: "exists false on refID when empty in Container",
			obj:  container,
			criteria: Query{
				Expr: ExistsExpr{Property: "@refID", Exists: false},
			},
			want: true,
		},
		{
			name: "exists on id",
			obj:  item,
			criteria: Query{
				Expr: ExistsExpr{Property: "@id", Exists: true},
			},
			want: true,
		},
		{
			name: "unknown property in binary expr returns false",
			obj:  item,
			criteria: Query{
				Expr: BinaryExpr{Property: "unknown:prop", Op: Equal, Operand: "foo"},
			},
			want: false,
		},
		{
			name: "unknown property in exists expr returns false",
			obj:  item,
			criteria: Query{
				Expr: ExistsExpr{Property: "unknown:prop", Exists: true},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Matches(tt.obj, tt.criteria)
			if got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClassIsDerivedFrom(t *testing.T) {
	tests := []struct {
		class    upnpav.Class
		ancestor upnpav.Class
		want     bool
	}{
		{
			class:    "object.item.videoItem.movie",
			ancestor: "object.item.videoItem",
			want:     true,
		},
		{
			class:    "object.item.videoItem",
			ancestor: "object.item.videoItem",
			want:     true,
		},
		{
			class:    "object.item.audioItem",
			ancestor: "object.item.videoItem",
			want:     false,
		},
		{
			class:    "object.item",
			ancestor: "object.item.audioItem",
			want:     false,
		},
	}

	for _, tt := range tests {
		got := ClassIsDerivedFrom(tt.class, tt.ancestor)
		if got != tt.want {
			t.Errorf("ClassIsDerivedFrom(%s, %s) = %v, want %v", tt.class, tt.ancestor, got, tt.want)
		}
	}
}

package maml

import (
	"reflect"
	"testing"
)

func TestUnmarshal_EmbeddedStructs(t *testing.T) {
	tests := []struct {
		name      string
		mamlInput string
		target    any
		expected  any
		wantErr   bool
	}{
		{
			name: "Basic embedded struct (value)",
			mamlInput: `
				{
					Name: "John Doe"
					City: "New York"
					PostalCode: "10001"
				}
			`,
			target: &struct {
				Name string
				Address
			}{},
			expected: &struct {
				Name string
				Address
			}{
				Name: "John Doe",
				Address: Address{
					City:       "New York",
					PostalCode: "10001",
				},
			},
			wantErr: false,
		},
		{
			name: "Basic embedded struct (pointer)",
			mamlInput: `
				{
					Name: "Jane Doe"
					City: "London"
					PostalCode: "SW1A 0AA"
				}
			`,
			target: &struct {
				Name string
				*Address
			}{},
			expected: &struct {
				Name string
				*Address
			}{
				Name: "Jane Doe",
				Address: &Address{
					City:       "London",
					PostalCode: "SW1A 0AA",
				},
			},
			wantErr: false,
		},
		{
			name: "Embedded struct with maml tags",
			mamlInput: `
				{
					User: "Alice"
					homeCity: "Paris"
				}
			`,
			target: &struct {
				User string
				TaggedAddress
			}{},
			expected: &struct {
				User string
				TaggedAddress
			}{
				User: "Alice",
				TaggedAddress: TaggedAddress{
					City: "Paris",
				},
			},
			wantErr: false,
		},
		{
			name: "Field shadowing by outer struct (same type)",
			mamlInput: `
				{
					Name: "Shadowed Name"
					City: "Outer City"
					PostalCode: "99999"
				}
			`,
			target: &struct {
				City string
				Address
			}{},
			expected: &struct {
				City string
				Address
			}{
				City: "Outer City",
				Address: Address{
					City:       "", // Should be shadowed
					PostalCode: "99999",
				},
			},
			wantErr: false,
		},
		{
			name: "Field shadowing by outer struct (different type)",
			mamlInput: `
				{
					ID: "outer-id"
					Name: "Bob"
					City: "Berlin"
				}
			`,
			target: &struct {
				ID string
				UserWithID
			}{},
			expected: &struct {
				ID string
				UserWithID
			}{
				ID: "outer-id",
				UserWithID: UserWithID{
					ID:   0, // Should be shadowed
					Name: "Bob",
				},
			},
			wantErr: false,
		},
		{
			name: "Nested embedded structs",
			mamlInput: `
				{
					Name: "Charlie"
					City: "Rome"
					PostalCode: "00100"
					CountryName: "Italy"
				}
			`,
			target: &struct {
				Name string
				DetailedAddress
			}{},
			expected: &struct {
				Name string
				DetailedAddress
			}{
				Name: "Charlie",
				DetailedAddress: DetailedAddress{
					Address: Address{
						City:       "Rome",
						PostalCode: "00100",
					},
					Country: Country{
						Name: "Italy",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Nested embedded structs with pointer",
			mamlInput: `
				{
					Name: "David"
					City: "Tokyo"
					PostalCode: "100-0001"
					CountryName: "Japan"
				}
			`,
			target: &struct {
				Name string
				*DetailedAddress
			}{},
			expected: &struct {
				Name string
				*DetailedAddress
			}{
				Name: "David",
				DetailedAddress: &DetailedAddress{
					Address: Address{
						City:       "Tokyo",
						PostalCode: "100-0001",
					},
					Country: Country{
						Name: "Japan",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Embedded struct field with no corresponding input",
			mamlInput: `
				{
					Name: "Eve"
				}
			`,
			target: &struct {
				Name string
				Address
			}{},
			expected: &struct {
				Name string
				Address
			}{
				Name: "Eve",
				Address: Address{
					City:       "",
					PostalCode: "",
				},
			},
			wantErr: false,
		},
		{
			name: "Outer struct field with no corresponding input",
			mamlInput: `
				{
					City: "Sydney"
					PostalCode: "2000"
				}
			`,
			target: &struct {
				Name string
				Address
			}{},
			expected: &struct {
				Name string
				Address
			}{
				Name: "",
				Address: Address{
					City:       "Sydney",
					PostalCode: "2000",
				},
			},
			wantErr: false,
		},
		{
			name: "Multiple embedded structs, no name collision",
			mamlInput: `
				{
					Name: "Frank"
					City: "Dublin"
					Street: "O'Connell St"
					Website: "example.com"
				}
			`,
			target: &struct {
				Name string
				Address
				ContactInfo
			}{},
			expected: &struct {
				Name string
				Address
				ContactInfo
			}{
				Name: "Frank",
				Address: Address{
					City:       "Dublin",
					PostalCode: "", // Not in input
				},
				ContactInfo: ContactInfo{
					Street:  "O'Connell St",
					Website: "example.com",
				},
			},
			wantErr: false,
		},
		{
			name: "Multiple embedded structs, with name collision, shallower takes precedence",
			mamlInput: `
				{
					Name: "Grace"
					City: "Edinburgh"
					CommonField: "outer value"
				}
			`,
			target: &struct {
				Name        string
				CommonField string // This should take precedence
				Embedded1
				Embedded2
			}{},
			expected: &struct {
				Name        string
				CommonField string
				Embedded1
				Embedded2
			}{
				Name:        "Grace",
				CommonField: "outer value",
				Embedded1: Embedded1{
					CommonField: "", // Shadowed by outer
				},
				Embedded2: Embedded2{
					CommonField: "", // Shadowed by outer
				},
			},
			wantErr: false,
		},
		{
			name: "Multiple embedded structs, with name collision, first declared takes precedence",
			mamlInput: `
				{
					Name: "Heidi"
					City: "Oslo"
					CommonField: "embedded1 value"
				}
			`,
			target: &struct {
				Name      string
				Embedded1 // This should take precedence at same depth
				Embedded2
			}{},
			expected: &struct {
				Name string
				Embedded1
				Embedded2
			}{
				Name: "Heidi",
				Embedded1: Embedded1{
					CommonField: "embedded1 value",
				},
				Embedded2: Embedded2{
					CommonField: "", // Shadowed by Embedded1
				},
			},
			wantErr: false,
		},
		{
			name: "MAML input with case-insensitive matching for embedded fields",
			mamlInput: `
				{
					Name: "Ivan"
					city: "Helsinki"
					POSTALCODE: "00100"
				}
			`,
			target: &struct {
				Name string
				Address
			}{},
			expected: &struct {
				Name string
				Address
			}{
				Name: "Ivan",
				Address: Address{
					City:       "Helsinki",
					PostalCode: "00100",
				},
			},
			wantErr: false,
		},
		{
			name: "MAML input with mixed case and tag precedence for embedded fields",
			mamlInput: `
				{
					User: "Julia"
					homecity: "Stockholm"
					postalCode: "11187"
				}
			`,
			target: &struct {
				User string
				TaggedAddress
			}{},
			expected: &struct {
				User string
				TaggedAddress
			}{
				User: "Julia",
				TaggedAddress: TaggedAddress{
					City:       "Stockholm",
					PostalCode: "11187",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Unmarshal([]byte(tt.mamlInput), tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(tt.target, tt.expected) {
				t.Errorf("Unmarshal() got = %v, want %v", tt.target, tt.expected)
			}
		})
	}
}

// Helper structs for testing
type Address struct {
	City       string
	PostalCode string
}

type TaggedAddress struct {
	City       string `maml:"homeCity"`
	PostalCode string `maml:"postalCode"`
}

type UserWithID struct {
	ID   int
	Name string
}

type Country struct {
	Name string `maml:"countryName"`
}

type DetailedAddress struct {
	Address
	Country
}

type ContactInfo struct {
	Street  string
	Website string
}

type Embedded1 struct {
	CommonField string
}

type Embedded2 struct {
	CommonField string
}

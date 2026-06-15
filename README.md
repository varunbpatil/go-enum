# go-enum

A zero-dependency code generator for Go enums that produces exported types, constants, and helper methods from **unexported** `iota` declarations. The generator itself uses only the Go standard library, and the generated code has no external dependencies either.

## Why not goenums?

[goenums](https://github.com/zarldev/goenums) wraps each enum variant in a struct, which breaks exhaustiveness checking with the [`exhaustive`](https://github.com/nishanths/exhaustive) linter. go-enum keeps variants as plain `const` values so switch/map exhaustiveness works correctly:

```go
switch status {
case OrderStatusPending:    // linter knows all cases
case OrderStatusConfirmed:
case OrderStatusShipped:
// missing case → linter error
}
```

go-enum intentionally does not support custom fields on enum variants. [goenums](https://github.com/zarldev/goenums) allows attaching fields to each variant via the struct wrapper, but this only supports primitive types so that the struct wrapper is _comparable_. go-enum's approach is simpler and more powerful: write a plain mapping function with a switch, and the `exhaustive` linter ensures you never miss a case:

```go
func (s OrderStatus) Label() string {
    switch s {
    case OrderStatusPending:   return "Awaiting payment"
    case OrderStatusConfirmed: return "Payment confirmed"
    case OrderStatusShipped:   return "On the way"
    // missing case → linter error
    }
}
```

This works with any return type — not just primitives.

## Installation

```sh
go install github.com/varunbpatil/go-enum@latest
```

Or with Nix:

```sh
nix run github:varunbpatil/go-enum
```

## Usage

Define an **unexported** enum in a `.go` file and add a `go:generate` directive:

```go
//go:generate go-enum orderstatus.go

type orderStatus int

const (
    orderStatusUnknown    orderStatus = iota // invalid unknown
    orderStatusPending                       // pending
    orderStatusConfirmed                     // confirmed
    orderStatusShipped                       // shipped
    orderStatusDelivered                     // delivered
    orderStatusCancelled                     // cancelled
)
```

Run the generator:

```sh
go generate ./...
# or directly:
go-enum orderstatus.go
```

This writes `orderstatus.enums.go` alongside the input file.

## Input format

**Type declaration** — unexported, with an integer underlying type:

```go
type orderStatus int   // int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64
```

**Const block** — must start with `iota`. The line comment controls the string name and validity:

| Comment form               | String name       | IsValid |
|----------------------------|-------------------|---------|
| `// foo`                   | `"foo"`           | `true`  |
| `// "bank transfer"`       | `"bank transfer"` | `true`  |
| `// invalid`               | derived from name | `false` |
| `// invalid foo`           | `"foo"`           | `false` |
| `// invalid "not valid"`   | `"not valid"`     | `false` |
| *(no comment)*             | derived from name | `true`  |

String name derivation strips the type prefix and lowercases the first letter: `orderStatusPending` → `"pending"`.

Multiple enum types in the same file are all processed into a single output file.

## Generated output

For each enum type `fooBar`, the generator produces `FooBar` (exported) with:

| Name                                              | Description                                                               |
|---------------------------------------------------|---------------------------------------------------------------------------|
| `type FooBar int`                                 | Exported type alias                                                       |
| `const FooBarX FooBar = iota`                     | Exported constants                                                        |
| `(v FooBar) String() string`                      | `fmt.Stringer` — returns the comment-derived name                         |
| `(v FooBar) IsValid() bool`                       | False for values marked `invalid` or undefined                            |
| `ParseFooBar(any) (FooBar, error)`                | Parses string, `[]byte`, numeric types, `fmt.Stringer`, or the type itself|
| `AllFooBars() iter.Seq[FooBar]`                   | Iterator over all valid values (Go 1.23+)                                 |
| `ExhaustiveFooBars(func(FooBar))`                 | Calls f for every valid value                                             |
| `(v FooBar) MarshalJSON() / UnmarshalJSON()`      | JSON serialisation                                                        |
| `(v FooBar) MarshalText() / UnmarshalText()`      | `encoding.TextMarshaler/Unmarshaler`                                      |
| `(v FooBar) MarshalBinary() / UnmarshalBinary()`  | `encoding.BinaryMarshaler/Unmarshaler`                                    |
| `(v FooBar) MarshalYAML() / UnmarshalYAML()`      | yaml.v3-compatible                                                        |
| `(v *FooBar) Scan(any) error`                     | `database/sql.Scanner`                                                    |
| `(v FooBar) Value() (driver.Value, error)`        | `database/sql/driver.Valuer`                                              |

A `func _()` compile-time check is also emitted. It references the original unexported constants so the compiler errors if the source enum changes without re-running the generator.

## Example

Input (`examples/orderstatus.go`):

```go
package examples

//go:generate go-enum orderstatus.go

type orderStatus int

const (
    orderStatusUnknown    orderStatus = iota // invalid unknown
    orderStatusPending                       // pending
    orderStatusConfirmed                     // confirmed
    orderStatusShipped                       // shipped
)

type paymentMethod int

const (
    paymentMethodUnknown     paymentMethod = iota // invalid unknown
    paymentMethodCard                             // card
    paymentMethodBankTransfer                     // "bank transfer"
)
```

Generated (`examples/orderstatus.enums.go`, trimmed):

```go
// Code generated by go-enum. DO NOT EDIT.

package examples

type OrderStatus int

const (
    OrderStatusUnknown    OrderStatus = iota
    OrderStatusPending
    OrderStatusConfirmed
    OrderStatusShipped
)

func (v OrderStatus) String() string { ... }
func (v OrderStatus) IsValid() bool  { ... }
func ParseOrderStatus(input any) (OrderStatus, error) { ... }
func AllOrderStatuses() iter.Seq[OrderStatus] { ... }
func ExhaustiveOrderStatuses(f func(OrderStatus)) { ... }
// + JSON, Text, Binary, YAML, SQL interfaces

type PaymentMethod int
// ... same methods
```

Usage:

```go
s := OrderStatusShipped
fmt.Println(s)           // "shipped"
fmt.Println(s.IsValid()) // true

v, err := ParseOrderStatus("confirmed")
// v == OrderStatusConfirmed

for s := range AllOrderStatuses() {
    fmt.Println(s) // pending, confirmed, shipped
}

b, _ := json.Marshal(OrderStatusPending) // "pending"
```

## Requirements

Go 1.23+ (uses `iter.Seq`).

## Acknowledgements

Inspired by [goenums](https://github.com/zarldev/goenums) by [@zarldev](https://github.com/zarldev). The generated helper methods (marshaling interfaces, SQL scanner/valuer, parse functions, iteration helpers) follow the same API surface as goenums. The key difference is that go-enum does not wrap variants in a struct, preserving exhaustiveness checking with the `exhaustive` linter.

## License

MIT — see [LICENSE](LICENSE).

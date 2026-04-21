package examples

//go:generate go-enum orderstatus.go

// orderStatus represents the lifecycle state of a customer order.
type orderStatus int

const (
	orderStatusUnknown    orderStatus = iota // invalid unknown
	orderStatusPending                       // pending
	orderStatusConfirmed                     // confirmed
	orderStatusProcessing                    // processing
	orderStatusShipped                       // shipped
	orderStatusDelivered                     // delivered
	orderStatusCancelled                     // cancelled
	orderStatusRefunded                      // refunded
)

// paymentMethod represents how an order was paid.
type paymentMethod int

const (
	paymentMethodUnknown      paymentMethod = iota // invalid unknown
	paymentMethodCard                              // card
	paymentMethodBankTransfer                      // "bank transfer"
	paymentMethodWallet                            // wallet
	paymentMethodCrypto                            // crypto
)

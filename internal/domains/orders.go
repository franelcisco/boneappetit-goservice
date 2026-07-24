package domains

import "errors"

// OrderNotFound is returned when an order is not found in the database
var OrderNotFound = errors.New("order not found")

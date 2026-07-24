package shopify

// GetOrderByID is the GraphQL query for retrieving an order by its ID
const getOrderByIDQuery = `
	query orderByIDQuery($id: ID!) {
		order(id: $id) {
			id
			name
      tags
			createdAt
			displayFinancialStatus
			displayFulfillmentStatus
      paymentGatewayNames
      shippingLine {
        title
        id
        isRemoved
      }
			totalPriceSet {
				shopMoney {
					amount
					currencyCode
				}
			}
			lineItems(first: 5) {
				edges {
					node {
            title
						name
						quantity
						sku
            variant {
              title
              id
            }
					}
				}
			}
      customAttributes {
        key
        value
      }
			customer {
				displayName
				id
        email
        defaultPhoneNumber {
          phoneNumber
        }
        defaultAddress {
          address1
          address2
          city
          province
          country
          zip
          phone
        }
				parentId: metafield(namespace: "customer_fields", key: "parent_id") {
          key
          value
          jsonValue
        }
        directDebitAccount: metafield(namespace: "custom", key: "direct_debit_account") {
          key
          value
          jsonValue
        }
			}
		}
	}
`

const getOrderByName = `
query orderByName($query: String!, $first: Int!) {
  orders(first: $first, query: $query) {
    nodes {
			id
			name
      tags
			createdAt
			displayFinancialStatus
			displayFulfillmentStatus
      paymentGatewayNames
      shippingLine {
        title
        id
        isRemoved
      }
			totalPriceSet {
				shopMoney {
					amount
					currencyCode
				}
			}
			lineItems(first: 5) {
				edges {
					node {
            title
						name
						quantity
						sku
            variant {
              title
              id
            }
					}
				}
			}
      customAttributes {
        key
        value
      }
			customer {
				displayName
				id
        email
        defaultPhoneNumber {
          phoneNumber
        }
        defaultAddress {
          address1
          address2
          city
          province
          country
          zip
          phone
        }
				parentId: metafield(namespace: "customer_fields", key: "parent_id") {
          key
          value
          jsonValue
        }
        directDebitAccount: metafield(namespace: "custom", key: "direct_debit_account") {
          key
          value
          jsonValue
        }
			}
		}
    pageInfo {
      hasNextPage
      startCursor
      hasPreviousPage
      endCursor
    }
  }
}`

const setCustomerMetafield = `mutation UpdateCustomerParentID($id: ID!, $namespace: String!, $key: String!, $value: String!) {
  customerUpdate(input: {
    id: $id
    metafields: [
      {
        namespace: $namespace
        key: $key
        value: $value
      }
    ]
  }) {
    customer {
      id
    }
    userErrors {
      message
      field
    }
  }
}`

const getCustomerMetafield = `
query($id: ID!, $namespace: String!, $key: String!) {
  customer(id: $id) {
    metafield(namespace: $namespace, key: $key) { 
      key
      value
      jsonValue
    }
  }
}`

const markOrderAsPaid = `
mutation orderMarkAsPaid($id: ID!) {
  orderMarkAsPaid(input: { id: $id })
  {
    order {
      id
    }
    userErrors {
      field
      message
    }
  }
}
`

const createDraftOrder = `
mutation draftOrderCreate($input: DraftOrderInput!) {
  draftOrderCreate(input: $input) {
    draftOrder {
      id
      name
      invoiceUrl
    }
    userErrors {
      field
      message
    }
  }
}
`

const getDraftOrderByID = `
query draftOrderByID($id: ID!) {
  draftOrder(id: $id) {
    id
    name
    invoiceUrl
    totalPriceSet {
      shopMoney {
        amount
        currencyCode
      }
    }
    order {
			id
			name
      tags
			createdAt
			displayFinancialStatus
			displayFulfillmentStatus
      shippingLine {
        title
        id
        isRemoved
      }
			totalPriceSet {
				shopMoney {
					amount
					currencyCode
				}
			}
			lineItems(first: 5) {
				edges {
					node {
            title
						name
						quantity
						sku
            variant {
              id
            }
					}
				}
			}
      customAttributes {
        key
        value
      }
		}
    customer {
				displayName
				id
        defaultPhoneNumber {
          phoneNumber
        }
        defaultAddress {
          address1
          address2
          city
          province
          country
          zip
          phone
        }
				parentId: metafield(namespace: "customer_fields", key: "parent_id") {
          key
          value
          jsonValue
        }
        directDebitAccount: metafield(namespace: "custom", key: "direct_debit_account") {
          key
          value
          jsonValue
        }
			}
  }
}
`
const createOrder = `
mutation orderCreate($order: OrderCreateOrderInput!, $options: OrderCreateOptionsInput) {
  orderCreate(order: $order, options: $options) {
    order {
      id
      name
      statusPageUrl
	  totalPriceSet {
		shopMoney {
			amount
			currencyCode
		}
	  }
    }
    userErrors {
      field
      message
    }
  }
}
`

const draftOrderComplete = `
mutation DraftOrderComplete($id: ID!) {
  draftOrderComplete(id: $id) {
    draftOrder {
      id
      status
      order {
        id
        name
        displayFinancialStatus
      }
    }
    userErrors {
      field
      message
    }
  }
}`


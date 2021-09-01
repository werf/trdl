package util

import (
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

func CheckRequiredFields(req *logical.Request, fields *framework.FieldData) *logical.Response {
	for fieldName, schema := range fields.Schema {
		if schema.Required && req.Get(fieldName) == nil {
			return logical.ErrorResponse("Required field %q must be set", fieldName)
		}
	}

	return nil
}

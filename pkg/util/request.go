package util

import (
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

func ValidateRequestFields(req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	fields.Raw = req.Data
	if err := fields.Validate(); err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	for fieldName, schema := range fields.Schema {
		if schema.Required && req.Get(fieldName) == nil {
			return logical.ErrorResponse(fmt.Sprintf("required field %q must be set", fieldName)), nil
		}
	}

	return nil, nil
}

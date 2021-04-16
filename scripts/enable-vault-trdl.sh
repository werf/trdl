#!/bin/bash

export VAULT_ADDR='http://127.0.0.1:8200'

vault secrets enable -path=trdl vault-plugin-secrets-trdl

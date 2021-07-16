package pgp

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type pathConfigureTrustedPGPPublicKeyCallbacksSuite struct {
	suite.Suite
	ctx     context.Context
	backend logical.Backend
	req     *logical.Request
	storage logical.Storage
}

func (suite *pathConfigureTrustedPGPPublicKeyCallbacksSuite) SetupTest() {
	ctx := context.Background()
	b := &framework.Backend{}
	b.Paths = Paths()
	storage := &logical.InmemStorage{}
	config := logical.TestBackendConfig()
	config.StorageView = storage
	err := b.Setup(ctx, config)
	assert.Nil(suite.T(), err)

	suite.ctx = ctx
	suite.backend = b
	suite.req = &logical.Request{Storage: storage}
	suite.storage = storage
}

func (suite *pathConfigureTrustedPGPPublicKeyCallbacksSuite) TestKeyCreateOrUpdate_SeveralKeys() {
	suite.req.Path = "configure/trusted_pgp_public_key"
	suite.req.Operation = logical.CreateOperation

	for _, reqDataKey := range []map[string]interface{}{
		dataTrustedPGPPublicKey1(),
		dataTrustedPGPPublicKey2(),
	} {
		suite.req.Data = reqDataKey
		resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
		assert.Nil(suite.T(), err)
		assert.Nil(suite.T(), resp)
	}

	keys, err := GetTrustedPGPPublicKeys(suite.ctx, suite.storage)
	assert.Nil(suite.T(), err)

	for _, reqDataKey := range []map[string]interface{}{
		dataTrustedPGPPublicKey1(),
		dataTrustedPGPPublicKey2(),
	} {
		assert.Contains(suite.T(), keys, reqDataKey[fieldNameTrustedPGPPublicKeyData])
	}
}

func (suite *pathConfigureTrustedPGPPublicKeyCallbacksSuite) TestKeyCreateOrUpdate_RequiredFields() {
	suite.req.Path = "configure/trusted_pgp_public_key"
	suite.req.Operation = logical.CreateOperation

	for _, fieldName := range []string{fieldNameTrustedPGPPublicKeyName, fieldNameTrustedPGPPublicKeyData} {
		suite.Run(fieldName, func() {
			data := dataTrustedPGPPublicKey1()
			delete(data, fieldName)

			suite.req.Data = data

			resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
			assert.Nil(suite.T(), err)
			assert.Equal(suite.T(), logical.ErrorResponse("Required field %q must be set", fieldName), resp)
		})
	}
}

func (suite *pathConfigureTrustedPGPPublicKeyCallbacksSuite) TestReadOrList_NoKeys() {
	suite.req.Path = "configure/trusted_pgp_public_key"
	suite.req.Operation = logical.ListOperation

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), logical.ListResponse([]string(nil)), resp)
}

func (suite *pathConfigureTrustedPGPPublicKeyCallbacksSuite) TestReadOrList_Keys() {
	var expectedDataKeys []string
	for _, reqDataKey := range []map[string]interface{}{
		dataTrustedPGPPublicKey1(),
		dataTrustedPGPPublicKey2(),
	} {
		keyName := reqDataKey[fieldNameTrustedPGPPublicKeyName].(string)
		keyData := reqDataKey[fieldNameTrustedPGPPublicKeyData].(string)
		err := suite.storage.Put(suite.ctx, &logical.StorageEntry{
			Key:   trustedPGPPublicKeyStorageKey(keyName),
			Value: []byte(keyData),
		})
		assert.Nil(suite.T(), err)

		expectedDataKeys = append(expectedDataKeys, keyName)
	}

	suite.req.Path = "configure/trusted_pgp_public_key"
	suite.req.Operation = logical.ListOperation

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), logical.ListResponse(expectedDataKeys), resp)
}

func (suite *pathConfigureTrustedPGPPublicKeyCallbacksSuite) TestKeyRead() {
	testData := dataTrustedPGPPublicKey1()
	testKeyName := testData[fieldNameTrustedPGPPublicKeyName].(string)
	testKeyData := testData[fieldNameTrustedPGPPublicKeyData].(string)
	err := suite.storage.Put(suite.ctx, &logical.StorageEntry{
		Key:   trustedPGPPublicKeyStorageKey(testKeyName),
		Value: []byte(testKeyData),
	})
	assert.Nil(suite.T(), err)

	suite.req.Path = fmt.Sprintf("configure/trusted_pgp_public_key/%s", testKeyName)
	suite.req.Operation = logical.ReadOperation

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(suite.T(), err)
	if assert.NotNil(suite.T(), resp) && assert.NotNil(suite.T(), resp.Data) {
		assert.Equal(
			suite.T(),
			map[string]interface{}{
				fieldNameTrustedPGPPublicKeyName: testKeyName,
				fieldNameTrustedPGPPublicKeyData: testKeyData,
			},
			resp.Data,
		)
	}
}

func (suite *pathConfigureTrustedPGPPublicKeyCallbacksSuite) TestKeyRead_NoKey() {
	testKeyName := "key_name"

	suite.req.Path = fmt.Sprintf("configure/trusted_pgp_public_key/%s", testKeyName)
	suite.req.Operation = logical.ReadOperation

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), logical.ErrorResponse("PGP public key %q not found in storage", testKeyName), resp)
}

func (suite *pathConfigureTrustedPGPPublicKeyCallbacksSuite) TestKeyDelete() {
	testData := dataTrustedPGPPublicKey1()
	testKeyName := testData[fieldNameTrustedPGPPublicKeyName].(string)
	testKeyData := testData[fieldNameTrustedPGPPublicKeyData].(string)
	err := suite.storage.Put(suite.ctx, &logical.StorageEntry{
		Key:   trustedPGPPublicKeyStorageKey(testKeyName),
		Value: []byte(testKeyData),
	})
	assert.Nil(suite.T(), err)

	suite.req.Path = fmt.Sprintf("configure/trusted_pgp_public_key/%s", testKeyName)
	suite.req.Operation = logical.DeleteOperation

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(suite.T(), err)
	assert.Nil(suite.T(), resp)

	entry, err := suite.storage.Get(suite.ctx, trustedPGPPublicKeyStorageKey(testKeyName))
	assert.Nil(suite.T(), err)
	assert.Nil(suite.T(), entry)
}

func (suite *pathConfigureTrustedPGPPublicKeyCallbacksSuite) TestKeyDelete_NoKey() {
	testKeyName := "key_name"

	suite.req.Path = fmt.Sprintf("configure/trusted_pgp_public_key/%s", testKeyName)
	suite.req.Operation = logical.DeleteOperation

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(suite.T(), err)
	assert.Nil(suite.T(), resp)
}

func TestBackendPathConfigureTrustedPGPPublicKeyCallbacks(t *testing.T) {
	suite.Run(t, new(pathConfigureTrustedPGPPublicKeyCallbacksSuite))
}

func dataTrustedPGPPublicKey1() map[string]interface{} {
	return map[string]interface{}{
		fieldNameTrustedPGPPublicKeyName: "my_key_1",
		fieldNameTrustedPGPPublicKeyData: `-----BEGIN PGP PUBLIC KEY BLOCK-----

mQINBFRUAGoBEACuk6ze2V2pZtScf1Ul25N2CX19AeL7sVYwnyrTYuWdG2FmJx4x
DLTLVUazp2AEm/JhskulL/7VCZPyg7ynf+o20Tu9/6zUD7p0rnQA2k3Dz+7dKHHh
eEsIl5EZyFy1XodhUnEIjel2nGe6f1OO7Dr3UIEQw5JnkZyqMcbLCu9sM2twFyfa
WPYip7FgAjyhHYp7TVvWQAHAMli3CwTqYKpEFwH+cnb4Vij6jc+iRj2wDGEqxX9X
WkOBQ1KDzaqKuooBcao9jk5hhJxdD5TWiSlMon+UOEXq8DYO43ZIYKlu7mB91jyT
aMhACwonyiGI+dtYycEosSphLtnM2cZMPZ4uW94H0TEXUJEDFvLC88JkM3YIU97w
fvfidBGruUYC+mTw7CusaCOQbBuZBiYduFgH8hRW97KLmHn0xzB1FV++KI7syo8q
XGo8Un24WP40IT78XjKO
=nUop
-----END PGP PUBLIC KEY BLOCK-----`,
	}
}

func dataTrustedPGPPublicKey2() map[string]interface{} {
	return map[string]interface{}{
		fieldNameTrustedPGPPublicKeyName: "my_key_2",
		fieldNameTrustedPGPPublicKeyData: `-----BEGIN PGP PUBLIC KEY BLOCK-----

mQINBFRUAGoBEACuk6ze2V2pZtScf1Ul25N2CX19AeL7sVYwnyrTYuWdG2FmJx4x
WkOBQ1KDzaqKuooBcao9jk5hhJxdD5TWiSlMon+UOEXq8DYO43ZIYKlu7mB91jyT
DLTLVUazp2AEm/JhskulL/7VCZPyg7ynf+o20Tu9/6zUD7p0rnQA2k3Dz+7dKHHh
eEsIl5EZyFy1XodhUnEIjel2nGe6f1OO7Dr3UIEQw5JnkZyqMcbLCu9sM2twFyfa
WPYip7FgAjyhHYp7TVvWQAHAMli3CwTqYKpEFwH+cnb4Vij6jc+iRj2wDGEqxX9X
aMhACwonyiGI+dtYycEosSphLtnM2cZMPZ4uW94H0TEXUJEDFvLC88JkM3YIU97w
fvfidBGruUYC+mTw7CusaCOQbBuZBiYduFgH8hRW97KLmHn0xzB1FV++KI7syo8q
XGo8Un24WP40IT78XjKO
=nUop
-----END PGP PUBLIC KEY BLOCK-----`,
	}
}

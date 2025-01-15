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

mQGNBGH6xLwBDACmDGe0qiJ3jXAJFbuWVMV6yAhk0ube/qGtijnsbyAkSU9bG6DM
DWgIVY1C86KVBqQBnJpiIsWYTUbtmxjEgg+KgUCxHUYXXhiTBW6aD+7Mpj7mxQ3A
Zim/8pNAIPRtQHTODPpFFxekfO1XuFC+CPQv3/XsuVHv6rTKK9V+ScbVL0Et7Vc9
PuZJfhTSrKQUnL8AMsI4cpLObO68lee3uU70aGG1twd0kfwzKuTTODCYIxbMfpAS
cMiORMYyK/e94mZb1EK0qVuZTiOqhVFjBFcMBeRDnUzB4nM3wWiVOdA/2TItLxyG
4QnQ/BSzBJRumdaFvk26rgTcacdXFiNUviODhM8J12JOYAq8d75ipQ3wyPDwz2IJ
3ZoeNhq66UslMpdL7xWK/06IelPCk2WrSWU+NGmmR0wBu1pnHZwS64gwjakH0OgH
cAKa1UQPBcpC35yoxToWn+HpUBx+cehPfRyWP9F3CdkleJQ6UVvpfwU1uJgSqt0V
Wvdb7rz+4T3spMMAEQEAAbQeRGV2ZWxvcGVyIDxkZXZlbG9wZXJAdHJkbC5kZXY+
iQHOBBMBCgA4FiEEdOElkCmxR8tAM+i4DUycFA6KEDAFAmH6xLwCGwMFCwkIBwIG
FQoJCAsCBBYCAwECHgECF4AACgkQDUycFA6KEDANEQv9GkFZz2+/giuhY82RKpS1
doiNfMezGRnQqp73x6ot24/HwbCxDyrnfpGv145qIH9ApKFRGMNvQHpAWYEfWddo
nHo9kkR7qqVaKnR9/V7NzuyOKbI4rtB/1i9RQjz1JLctvGY/7WdA0SVDz+tPnSBw
/aIfa5nEgD20Oyqgd8qakHfyHFVmfMGQ27rDihuNOHuL1eDmschEeFRPa3uzKeIQ
tOuw0uw9jSDOLoHGUCe3SmV7oMJ+B4biDL7ZazZgTXD/fOvBN/SN5MVr7fbL/BcT
jWBxyPhUy1QvF6j9pA84LcsOA61MptVGslOw9l6oEzGWlYZMrZfhQEW4DX7LmfOc
F9SuZE9Usu1fVP//ljxwg5mEXtcdyeo3u57hIwot7Jbv/18R3Nx2o4u2WMbZA1u5
H13Ow4FLsqgdCEz8BxCp3luqJalIiViEn3Fl6CqpSdveaNya+EHhwAqLdlRapGTO
1DcACljS/ToUzD9GmmzEfMF+j9Cg0QV928nkhpWwO2l3uQGNBGH6xLwBDAC03NfW
m0+JgBAGse/xeiMBf7zmtuE3fbe0nW/YqC2MWCUiC3QMfNFUAz1tktev5HNUw2A4
0ON6DV8Lb5YqOOZqya+e2QR/Z50MF362895fYz2pske1oV8/D3t3lJk47Cb9s2TN
yD26yWp4vhessTutZmqPourEAddeicrJGoCPn6Dt/cyI0wW/vFwlTju7zhem/Lyx
vQSSBzKoKXFaG5xGlnT4WXLtNb85ePxrYLzcvAGYgmp3yF1EYeD3t9bdD/kmXu2P
5yBlZesYZJiF9Qw6Xvzvmcp8EsMURGCFLU4tk0k8Xs6gWyddtmhfhrj6OXmoVHZN
5pwIMzXoUtL765fnsqPiflIU521dTbk9Q/Kw9p6GnQ30Ebz1lkws9fefEkm2TdRN
ViJ/CwxgqquChXpYbo3fkeh5b/Z8pSgLXGJafRtuiD/keuc+Gg+2SpLHbvuBSzhp
cE/YUt7jYqvHC1la1gMWZbNuGePa2ICDDnonvo7vnprgQ3Z9+i2CwyZh2RUAEQEA
AYkBtgQYAQoAIBYhBHThJZApsUfLQDPouA1MnBQOihAwBQJh+sS8AhsMAAoJEA1M
nBQOihAwmpEL/RaECBsCa0yRcbldE972+w9kC7aEmlaS/k5P/v6b9QRHVKGO2CPO
ImdeeOwRWGxARU4LxjSBD3JjhK2YfKgBJqiIodeNDy7S06ORvTQfpQxpKZe66ySJ
FaUEE4rrb7F3IegnrkJ20mId10wn/exEFc/+H5UzzlXvbD29Ussq+3TXgtPHdrk9
qwTYDMlJpq4hGJVSRBcBSHKMMaEwPr/9qb82bd0yhRPdxVA7d29J1fcI3joCjDQy
L5fboMLUPyzfrv1VlILQZHaxvC5oATU9HfuGBdbze840p7DSYuekUpXYBgUlaIWC
R56SxbtJhHPwj8B/pqJX1LKDUHHF8rv1BqlHLy/iTulJn9pNlvWYaM1iWM1FnncZ
k2NYwYspTmI+WsmagXtueszb5p4exlCKyheT2/z1fvrWinOmU8ylsI0OA9FGXVma
eiX/1DGByT7JKMWA6P1+v+YXmHBdyoAYAoUdhRJFZoVKTC06PeZT8tOwMXeDZCdW
XaOlJrPDM5E9zw==
=bIYD
-----END PGP PUBLIC KEY BLOCK-----
`,
	}
}

func dataTrustedPGPPublicKey2() map[string]interface{} {
	return map[string]interface{}{
		fieldNameTrustedPGPPublicKeyName: "my_key_2",
		fieldNameTrustedPGPPublicKeyData: `-----BEGIN PGP PUBLIC KEY BLOCK-----

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
WPi4XYQTYGzk
=4KEN
-----END PGP PUBLIC KEY BLOCK-----`,
	}
}

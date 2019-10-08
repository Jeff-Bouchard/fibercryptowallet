package skycoin

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/skycoin/skycoin/src/api"
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/readable"
)

func TestTransactionFinderAddressesActivity(t *testing.T) {
	CleanGlobalMock()

	addresses := make([]cipher.Address, 0)
	addressesN := make([]string, 0)

	for i := 0; i < 3; i++ {
		p, _ := cipher.GenerateKeyPair()
		a := cipher.AddressFromPubKey(p)
		s := a.String()
		addresses = append(addresses, a)
		addressesN = append(addressesN, s)
	}

	global_mock.On("Transactions", []string{}).Return(nil, nil)
	global_mock.On("Transactions", []string{addressesN[0]}).Return(
		[]readable.TransactionWithStatus{},
		nil)
	global_mock.On("Transactions", []string{addressesN[1]}).Return(
		[]readable.TransactionWithStatus{
			readable.TransactionWithStatus{
				Status: readable.TransactionStatus{
					Confirmed: true,
				},
			},
		},
		nil)
	global_mock.On("Transactions", []string{addressesN[2]}).Return(
		[]readable.TransactionWithStatus{
			readable.TransactionWithStatus{
				Status: readable.TransactionStatus{
					Confirmed: true,
				},
			},
			readable.TransactionWithStatus{
				Status: readable.TransactionStatus{
					Confirmed: false,
				},
			},
		},
		nil)

	thxF := &TransactionFinder{}

	mask, err := thxF.AddressesActivity([]cipher.Address{})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(mask))
	assert.Equal(t, []bool{}, mask)

	mask, err = thxF.AddressesActivity(addresses)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(mask))
	assert.Equal(t, false, mask[0])
	for i := 1; i < 3; i++ {
		assert.Equal(t, true, mask[i])
	}
}

func TestSkycoinRemoteWalletListWallets(t *testing.T) {
	CleanGlobalMock()

	global_mock.On("Wallets").Return(
		[]api.WalletResponse{
			api.WalletResponse{
				Meta: readable.WalletMeta{
					Coin:      "Sky",
					Filename:  "FiberCrypto",
					Label:     "wallet",
					Encrypted: true,
				},
			},
			api.WalletResponse{
				Meta: readable.WalletMeta{
					Coin:      "Sky",
					Filename:  "FiberCrypto",
					Label:     "wallet",
					Encrypted: true,
				},
			},
		},
		nil)

	wltSrv := &SkycoinRemoteWallet{poolSection: "skycoin"}
	iter := wltSrv.ListWallets()
	for iter.Next() {
		wlt := iter.Value()
		assert.Equal(t, "wallet", wlt.GetLabel())
		assert.Equal(t, "FiberCrypto", wlt.GetId())
	}
}

func TestSkycoinRemoteWalletCreateWallet(t *testing.T) {
	CleanGlobalMock()

	seed, label, pwd, scanN := "seed", "label", "pwd", 666

	wltOpt1 := api.CreateWalletOptions{
		Type:     WalletTypeDeterministic,
		Seed:     seed,
		Label:    label,
		Password: pwd,
		ScanN:    scanN,
		Encrypt:  true,
	}
	wltOpt2 := api.CreateWalletOptions{
		Type:    WalletTypeDeterministic,
		Seed:    seed,
		Label:   label,
		ScanN:   scanN,
		Encrypt: false,
	}

	global_mock.On("CreateWallet", wltOpt1).Return(
		&api.WalletResponse{
			Meta: readable.WalletMeta{
				Coin:      "Sky",
				Filename:  "FiberCrypto",
				Label:     "walletEncrypted",
				Encrypted: true,
			},
		},
		nil)
	global_mock.On("CreateWallet", wltOpt2).Return(
		&api.WalletResponse{
			Meta: readable.WalletMeta{
				Coin:      "Sky",
				Filename:  "FiberCrypto",
				Label:     "walletNonEncrypted",
				Encrypted: false,
			},
		},
		nil)

	wltSrv := &SkycoinRemoteWallet{poolSection: "skycoin"}
	pwdReader := func(message string) (string, error) {
		return "pwd", nil
	}

	wlt1, err := wltSrv.CreateWallet(label, seed, true, pwdReader, scanN)
	assert.Nil(t, err)
	assert.Equal(t, "walletEncrypted", wlt1.GetLabel())
	assert.Equal(t, "FiberCrypto", wlt1.GetId())

	wlt2, err := wltSrv.CreateWallet(label, seed, false, pwdReader, scanN)
	assert.Nil(t, err)
	assert.Equal(t, "walletNonEncrypted", wlt2.GetLabel())
	assert.Equal(t, "FiberCrypto", wlt2.GetId())
}

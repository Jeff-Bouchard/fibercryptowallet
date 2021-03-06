package skycoin

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/SkycoinProject/skycoin/src/visor"

	"github.com/fibercrypto/fibercryptowallet/src/coin/mocks"
	"github.com/fibercrypto/fibercryptowallet/src/coin/skycoin/params"
	"github.com/fibercrypto/fibercryptowallet/src/core"
	"github.com/fibercrypto/fibercryptowallet/src/util"

	"github.com/SkycoinProject/skycoin/src/api"
	"github.com/SkycoinProject/skycoin/src/cipher"
	"github.com/SkycoinProject/skycoin/src/coin"
	"github.com/SkycoinProject/skycoin/src/readable"
	"github.com/SkycoinProject/skycoin/src/testutil"
	"github.com/SkycoinProject/skycoin/src/wallet"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

	mockSkyApiTransactions(global_mock, addressesN)

	thxF := &TransactionFinder{}

	mask, err := thxF.AddressesActivity([]cipher.Address{})
	require.NoError(t, err)
	require.Equal(t, 0, len(mask))
	require.Equal(t, []bool{}, mask)

	mask, err = thxF.AddressesActivity(addresses)
	require.NoError(t, err)
	require.Equal(t, 3, len(mask))
	require.Equal(t, false, mask[0])
	for i := 1; i < 3; i++ {
		require.Equal(t, true, mask[i])
	}
}

func TestSkycoinRemoteWalletListWallets(t *testing.T) {

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

	wltSrv := &SkycoinRemoteWallet{poolSection: PoolSection}
	iter := wltSrv.ListWallets()
	for iter.Next() {
		wlt := iter.Value()
		require.Equal(t, "wallet", wlt.GetLabel())
		require.Equal(t, "FiberCrypto", wlt.GetId())
	}
}

func TestSkycoinRemoteWalletCreateWallet(t *testing.T) {

	seed, label, pwd, scanN := "seed", "label", "pwd", 666

	wltOpt1 := api.CreateWalletOptions{
		Type:     wallet.WalletTypeDeterministic,
		Seed:     seed,
		Label:    label,
		Password: pwd,
		ScanN:    scanN,
		Encrypt:  true,
	}
	wltOpt2 := api.CreateWalletOptions{
		Type:    wallet.WalletTypeDeterministic,
		Seed:    seed,
		Label:   label,
		ScanN:   scanN,
		Encrypt: false,
	}

	mockSkyApiCreateWallet(global_mock, &wltOpt1, "walletEncrypted", true)
	mockSkyApiCreateWallet(global_mock, &wltOpt2, "walletNonEncrypted", false)

	wltSrv := &SkycoinRemoteWallet{poolSection: PoolSection}
	pwdReader := func(message string, _ core.KeyValueStore) (string, error) {
		return "pwd", nil
	}

	wlt1, err := wltSrv.CreateWallet(label, seed, wallet.WalletTypeDeterministic, true, pwdReader, scanN)
	require.NoError(t, err)
	require.Equal(t, "walletEncrypted", wlt1.GetLabel())
	require.Equal(t, "FiberCrypto", wlt1.GetId())

	wlt2, err := wltSrv.CreateWallet(label, seed, wallet.WalletTypeDeterministic, false, pwdReader, scanN)
	require.NoError(t, err)
	require.Equal(t, "walletNonEncrypted", wlt2.GetLabel())
	require.Equal(t, "FiberCrypto", wlt2.GetId())
}

func TestSkycoinRemoteWalletEncrypt(t *testing.T) {
	CleanGlobalMock()
	global_mock.On("EncryptWallet", "wallet", "pwd").Return(&api.WalletResponse{}, nil)

	wltSrv := &SkycoinRemoteWallet{poolSection: PoolSection}
	pwdReader := func(message string, _ core.KeyValueStore) (string, error) {
		return "pwd", nil
	}

	wltSrv.Encrypt("wallet", pwdReader)
}

func TestSkycoinRemoteWalletDecrypt(t *testing.T) {
	CleanGlobalMock()
	global_mock.On("DecryptWallet", "wallet", "pwd").Return(&api.WalletResponse{}, nil)

	wltSrv := &SkycoinRemoteWallet{poolSection: PoolSection}
	pwdReader := func(message string, _ core.KeyValueStore) (string, error) {
		return "pwd", nil
	}

	wltSrv.Decrypt("wallet", pwdReader)
}

func TestSkycoinRemoteWalletIsEncrypted(t *testing.T) {

	global_mock.On("Wallet", "encrypted").Return(
		&api.WalletResponse{
			Meta: readable.WalletMeta{
				Encrypted: true,
			},
		},
		nil)
	global_mock.On("Wallet", "nonEncrypted").Return(
		&api.WalletResponse{
			Meta: readable.WalletMeta{
				Encrypted: false,
			},
		},
		nil)

	wltSrv := &SkycoinRemoteWallet{poolSection: PoolSection}

	encrypted, err := wltSrv.IsEncrypted("encrypted")
	require.NoError(t, err)
	require.Equal(t, true, encrypted)

	encrypted, err = wltSrv.IsEncrypted("nonEncrypted")
	require.NoError(t, err)
	require.Equal(t, false, encrypted)
}

func TestSkycoinRemoteWalletGetWallet(t *testing.T) {
	CleanGlobalMock()

	global_mock.On("Wallet", "wallet").Return(
		&api.WalletResponse{
			Meta: readable.WalletMeta{
				Coin:      "Sky",
				Filename:  "FiberCrypto",
				Label:     "wallet",
				Encrypted: true,
			},
			Entries: []readable.WalletEntry{
				readable.WalletEntry{Address: "addr"},
			},
		},
		nil)

	wltSrv := &SkycoinRemoteWallet{poolSection: PoolSection}
	wlt := wltSrv.GetWallet("wallet")
	require.Equal(t, "wallet", wlt.GetLabel())
	require.Equal(t, "FiberCrypto", wlt.GetId())
}

func TestRemoteWalletSignSkycoinTxn(t *testing.T) {
	hash := testutil.RandSHA256(t)
	txn := coin.Transaction{
		Length:    100,
		Type:      0,
		InnerHash: hash,
	}
	unTxn := SkycoinUninjectedTransaction{
		txn:     &txn,
		inputs:  nil,
		outputs: nil,
		fee:     100,
	}
	encodedResponse, err := unTxn.txn.SerializeHex()
	require.NoError(t, err)

	walletSignTxn := api.WalletSignTransactionRequest{
		EncodedTransaction: encodedResponse,
		WalletID:           "wallet",
		Password:           "password",
		SignIndexes:        nil,
	}

	crtTxn, err := api.NewCreateTransactionResponse(&txn, nil)
	crtTxn.Transaction.Fee = "100"
	require.NoError(t, err)

	global_mock.On("WalletSignTransaction", walletSignTxn).Return(
		crtTxn,
		nil)

	wlt := &RemoteWallet{
		Id:          "wallet",
		Encrypted:   true,
		poolSection: PoolSection,
	}
	pwdReader := func(string, core.KeyValueStore) (string, error) {
		return "password", nil
	}
	ret, err := wlt.Sign(&unTxn, nil, pwdReader, nil)
	require.NoError(t, err)
	require.NotNil(t, ret)
	value, err := ret.ComputeFee(CoinHour)
	require.NoError(t, err)
	require.Equal(t, uint64(100), value)
}

func TestRemoteWalletSetLabel(t *testing.T) {
	CleanGlobalMock()
	global_mock.On("UpdateWallet", "walletId", "wallet").Return(nil)

	wlt := &RemoteWallet{
		Id:          "walletId",
		poolSection: PoolSection,
	}

	wlt.SetLabel("wallet")
}

func NewTransferOptions() *TransferOptions {
	tOptions := TransferOptions{
		values: make(map[string]interface{}),
	}
	return &tOptions
}

func TestRemoteWalletTransfer(t *testing.T) {
	CleanGlobalMock()
	destinationAddress := testutil.MakeAddress()
	sky := 500
	hash := testutil.RandSHA256(t)

	addr, err := NewSkycoinAddress(destinationAddress.String())
	require.NoError(t, err)
	opt := NewTransferOptions()
	opt.SetValue("BurnFactor", "0.5")
	opt.SetValue("CoinHoursSelectionType", "auto")

	req := api.CreateTransactionRequest{
		IgnoreUnconfirmed: false,
		HoursSelection: api.HoursSelection{
			Type:        "auto",
			Mode:        "share",
			ShareFactor: "0.5",
		},
		To: []api.Receiver{
			api.Receiver{
				Address: destinationAddress.String(),
				Coins:   strconv.Itoa(sky),
			},
		},
	}

	wreq := api.WalletCreateTransactionRequest{
		Unsigned:                 true,
		WalletID:                 "wallet",
		CreateTransactionRequest: req,
	}

	txn := coin.Transaction{
		Length:    100,
		Type:      0,
		InnerHash: hash,
	}
	crtTxn, err := api.NewCreateTransactionResponse(&txn, nil)
	require.NoError(t, err)
	crtTxn.Transaction.Fee = "500"

	mockSkyApiWalletCreateTransaction(global_mock, &wreq, crtTxn)

	wlt := &RemoteWallet{
		Id:          "wallet",
		poolSection: PoolSection,
	}
	quot, err := util.AltcoinQuotient(params.SkycoinTicker)
	require.NoError(t, err)

	destination := &SkycoinTransactionOutput{
		skyOut: readable.TransactionOutput{
			Address: addr.String(),
			Coins:   util.FormatCoins(uint64(sky*1e6), quot),
		}}

	ret, err := wlt.Transfer(destination, opt)
	require.NoError(t, err)
	require.NotNil(t, ret)
	val, err := ret.ComputeFee(CoinHour)
	require.NoError(t, err)
	require.Equal(t, uint64(sky), val)
	require.Equal(t, crtTxn.Transaction.TxID, ret.GetId())

}

func TestRemoteWalletSendFromAddress(t *testing.T) {
	CleanGlobalMock()
	startAddress := testutil.MakeAddress()
	destinationAddress := testutil.MakeAddress()
	changeAddress := (testutil.MakeAddress()).String()
	sky := 500
	hash := testutil.RandSHA256(t)

	toAddr := &SkycoinTransactionOutput{
		skyOut: readable.TransactionOutput{
			Address: destinationAddress.String(),
			Coins:   strconv.Itoa(sky),
			Hours:   uint64(250),
		},
	}
	fromAddr, err := NewSkycoinAddress(startAddress.String())
	require.NoError(t, err)
	chgAddr, err := NewSkycoinAddress(changeAddress)
	require.NoError(t, err)

	opt1 := NewTransferOptions()
	opt1.SetValue("BurnFactor", "0.5")
	opt1.SetValue("CoinHoursSelectionType", "auto")

	req1 := api.CreateTransactionRequest{
		IgnoreUnconfirmed: false,
		HoursSelection: api.HoursSelection{
			Type:        "auto",
			Mode:        "share",
			ShareFactor: "0.5",
		},
		ChangeAddress: &changeAddress,
		To: []api.Receiver{
			api.Receiver{
				Address: destinationAddress.String(),
				Coins:   strconv.Itoa(sky),
			},
		},
		Addresses: []string{startAddress.String()},
	}

	wreq1 := api.WalletCreateTransactionRequest{
		Unsigned:                 true,
		WalletID:                 "wallet1",
		CreateTransactionRequest: req1,
	}

	opt2 := NewTransferOptions()
	opt2.SetValue("BurnFactor", "0.5")
	opt2.SetValue("CoinHoursSelectionType", "manual")

	req2 := api.CreateTransactionRequest{
		IgnoreUnconfirmed: false,
		HoursSelection: api.HoursSelection{
			Type: "manual",
		},
		ChangeAddress: &changeAddress,
		To: []api.Receiver{
			api.Receiver{
				Address: destinationAddress.String(),
				Coins:   strconv.Itoa(sky),
				Hours:   "250",
			},
		},
		Addresses: []string{startAddress.String()},
	}

	wreq2 := api.WalletCreateTransactionRequest{
		Unsigned:                 true,
		WalletID:                 "wallet2",
		CreateTransactionRequest: req2,
	}

	txn := coin.Transaction{
		Length:    100,
		Type:      0,
		InnerHash: hash,
	}
	crtTxn, err := api.NewCreateTransactionResponse(&txn, nil)
	require.NoError(t, err)
	crtTxn.Transaction.Fee = strconv.Itoa(sky)

	mockSkyApiWalletCreateTransaction(global_mock, &wreq1, crtTxn)
	mockSkyApiWalletCreateTransaction(global_mock, &wreq2, crtTxn)

	// Testing HoursSelection to auto
	wlt1 := &RemoteWallet{
		Id:          "wallet1",
		poolSection: PoolSection,
	}

	ret, err := wlt1.SendFromAddress([]core.Address{&fromAddr}, []core.TransactionOutput{toAddr}, &chgAddr, opt1)
	require.NoError(t, err)
	require.NotNil(t, ret)
	val, err := ret.ComputeFee(CoinHour)
	require.NoError(t, err)
	require.Equal(t, util.FormatCoins(uint64(sky), 10), util.FormatCoins(uint64(val), 10))
	require.Equal(t, crtTxn.Transaction.TxID, ret.GetId())

	// Testing HoursSelection to manual
	wlt2 := &RemoteWallet{
		Id:          "wallet2",
		poolSection: PoolSection,
	}

	ret, err = wlt2.SendFromAddress([]core.Address{&fromAddr}, []core.TransactionOutput{toAddr}, &chgAddr, opt2)
	require.NoError(t, err)
	require.NotNil(t, ret)
	val, err = ret.ComputeFee(CoinHour)
	require.NoError(t, err)
	require.Equal(t, util.FormatCoins(uint64(sky), 10), util.FormatCoins(uint64(val), 10))
	require.Equal(t, crtTxn.Transaction.TxID, ret.GetId())
}

func TestRemoteWalletSpend(t *testing.T) {
	CleanGlobalMock()
	destinationAddress := testutil.MakeAddress()
	changeAddress := (testutil.MakeAddress()).String()
	sky := 500
	hash := testutil.RandSHA256(t)

	toAddr := &SkycoinTransactionOutput{
		skyOut: readable.TransactionOutput{
			Address: destinationAddress.String(),
			Coins:   strconv.Itoa(sky),
			Hours:   uint64(250),
		},
	}
	chgAddr, err := NewSkycoinAddress(changeAddress)
	require.NoError(t, err)
	opt := NewTransferOptions()
	opt.SetValue("BurnFactor", "0.5")
	opt.SetValue("CoinHoursSelectionType", "auto")

	req := api.CreateTransactionRequest{
		IgnoreUnconfirmed: false,
		HoursSelection: api.HoursSelection{
			Type:        "auto",
			Mode:        "share",
			ShareFactor: "0.5",
		},
		ChangeAddress: &changeAddress,
		To: []api.Receiver{
			api.Receiver{
				Address: destinationAddress.String(),
				Coins:   strconv.Itoa(sky),
			},
		},
	}

	wreq := api.WalletCreateTransactionRequest{
		Unsigned:                 true,
		WalletID:                 "wallet",
		CreateTransactionRequest: req,
	}

	txn := coin.Transaction{
		Length:    100,
		Type:      0,
		InnerHash: hash,
	}
	crtTxn, err := api.NewCreateTransactionResponse(&txn, nil)
	require.NoError(t, err)
	crtTxn.Transaction.Fee = "500"

	mockSkyApiWalletCreateTransaction(global_mock, &wreq, crtTxn)

	wlt := &RemoteWallet{
		Id:          "wallet",
		poolSection: PoolSection,
	}

	ret, err := wlt.Spend(nil, []core.TransactionOutput{toAddr}, &chgAddr, opt)
	require.NoError(t, err)
	require.NotNil(t, ret)
	val, err := ret.ComputeFee(CoinHour)
	require.NoError(t, err)
	require.Equal(t, uint64(sky), val)
	require.Equal(t, crtTxn.Transaction.TxID, ret.GetId())
}

func TestRemoteWalletGenAddresses(t *testing.T) {
	CleanGlobalMock()
	pwd := "pwd"

	global_mock.On("Wallet", "wallet").Return(
		&api.WalletResponse{
			Meta: readable.WalletMeta{
				Coin:      "Sky",
				Filename:  "FiberCrypto",
				Label:     "wallet",
				Encrypted: true,
			},
			Entries: []readable.WalletEntry{
				readable.WalletEntry{Address: "2JJ8pgq8EDAnrzf9xxBJapE2qkYLefW4uF8"},
			},
		},
		nil)

	global_mock.On("NewWalletAddress", "wallet", 1, pwd).Return(
		[]string{"2JJ8pgq8EDAnrzf9xxBJapE2qkYLefW4uF8", "2JJ8pgq8EDAnrzf9xxBJapE2qkYLefW4uF8"},
		nil)

	wlt := &RemoteWallet{
		Id:          "wallet",
		poolSection: PoolSection,
	}
	pwdReader := func(message string, _ core.KeyValueStore) (string, error) {
		return "pwd", nil
	}
	iter := wlt.GenAddresses(0, 0, 2, pwdReader)
	for iter.Next() {
		a := iter.Value()
		require.Equal(t, "2JJ8pgq8EDAnrzf9xxBJapE2qkYLefW4uF8", a.String())
	}
}

func TestRemoteWalletGetLoadedAddresses(t *testing.T) {

	wlt := &RemoteWallet{
		Id:          "wallet",
		poolSection: PoolSection,
	}
	iter, err := wlt.GetLoadedAddresses()
	require.NoError(t, err)
	items := 0
	for iter.Next() {
		a := iter.Value()
		items++
		require.Equal(t, "2JJ8pgq8EDAnrzf9xxBJapE2qkYLefW4uF8", a.String())
	}
	require.Equal(t, 1, items)
}

func TestUninjectedTransactionVerifySigned(t *testing.T) {
	// Mismatch header hash
	txn, err := makeTransaction(t)
	require.NoError(t, err)
	txn.InnerHash = cipher.SHA256{}
	uiTxn := makeUninjectedTransaction(t, &txn, 0)
	testutil.RequireError(t, uiTxn.VerifySigned(), "InnerHash does not match computed hash")

	// No inputs
	txn, err = makeTransaction(t)
	require.NoError(t, err)
	txn.In = make([]cipher.SHA256, 0)
	err = txn.UpdateHeader()
	require.NoError(t, err)
	uiTxn = makeUninjectedTransaction(t, &txn, 0)
	testutil.RequireError(t, uiTxn.VerifySigned(), "No inputs")

	// No outputs
	txn, err = makeTransaction(t)
	require.NoError(t, err)
	txn.Out = make([]coin.TransactionOutput, 0)
	err = txn.UpdateHeader()
	require.NoError(t, err)
	uiTxn = makeUninjectedTransaction(t, &txn, 0)
	testutil.RequireError(t, uiTxn.VerifySigned(), "No outputs")

	// Invalid number of sigs
	txn, err = makeTransaction(t)
	require.NoError(t, err)
	txn.Sigs = make([]cipher.Sig, 0)
	err = txn.UpdateHeader()
	require.NoError(t, err)
	uiTxn = makeUninjectedTransaction(t, &txn, 0)
	testutil.RequireError(t, uiTxn.VerifySigned(), "Invalid number of signatures")
	txn.Sigs = make([]cipher.Sig, 20)
	err = txn.UpdateHeader()
	require.NoError(t, err)
	testutil.RequireError(t, uiTxn.VerifySigned(), "Invalid number of signatures")

	// Too many sigs & inputs
	txn, err = makeTransaction(t)
	require.NoError(t, err)
	txn.Sigs = make([]cipher.Sig, math.MaxUint16+1)
	txn.In = make([]cipher.SHA256, math.MaxUint16+1)
	uiTxn = makeUninjectedTransaction(t, &txn, 0)
	testutil.RequireError(t, uiTxn.VerifySigned(), "Too many signatures and inputs")

	// Duplicate inputs
	ux, kd, err1 := makeUxOutWithSecret(t)
	require.NoError(t, err1)
	txn = makeTransactionFromUxOut(t, ux, kd.SecKey)
	err = txn.PushInput(txn.In[0])
	require.NoError(t, err)
	txn.Sigs = nil
	txn.SignInputs([]cipher.SecKey{kd.SecKey, kd.SecKey})
	err = txn.UpdateHeader()
	require.NoError(t, err)
	uiTxn = makeUninjectedTransaction(t, &txn, 0)
	testutil.RequireError(t, uiTxn.VerifySigned(), "Duplicate spend")

	// Duplicate outputs
	txn, err = makeTransaction(t)
	require.NoError(t, err)
	to := txn.Out[0]
	err = txn.PushOutput(to.Address, to.Coins, to.Hours)
	require.NoError(t, err)
	err = txn.UpdateHeader()
	require.NoError(t, err)
	uiTxn = makeUninjectedTransaction(t, &txn, 0)
	testutil.RequireError(t, uiTxn.VerifySigned(), "Duplicate output in transaction")

	// Invalid signature, empty
	txn, err = makeTransaction(t)
	require.NoError(t, err)
	txn.Sigs[0] = cipher.Sig{}
	uiTxn = makeUninjectedTransaction(t, &txn, 0)
	testutil.RequireError(t, uiTxn.VerifySigned(), "Unsigned input in transaction")

	// Invalid signature, not empty
	// A stable invalid signature must be used because random signatures could appear valid
	// Note: Transaction.Verify() only checks that the signature is a minimally valid signature
	badSig := "9a0f86874a4d9541f58a1de4db1c1b58765a868dc6f027445d0a2a8a7bddd1c45ea559fcd7bef45e1b76ccdaf8e50bbebd952acbbea87d1cb3f7a964bc89bf1ed5"
	txn, err = makeTransaction(t)
	require.NoError(t, err)
	txn.Sigs[0] = cipher.MustSigFromHex(badSig)
	uiTxn = makeUninjectedTransaction(t, &txn, 0)
	testutil.RequireError(t, uiTxn.VerifySigned(), "Failed to recover pubkey from signature")

	// We can't check here for other invalid signatures:
	//      - Signatures signed by someone else, spending coins they don't own
	//      - Signatures signing a different message
	// This must be done by blockchain tests, because we need the address
	// from the unspent being spent
	// The verification here only checks that the signature is valid at all

	// Output coins are 0
	txn, err = makeTransaction(t)
	require.NoError(t, err)
	txn.Out[0].Coins = 0
	err = txn.UpdateHeader()
	require.NoError(t, err)
	uiTxn = makeUninjectedTransaction(t, &txn, 0)
	testutil.RequireError(t, uiTxn.VerifySigned(), "Zero coin output")

	// Output coin overflow
	txn, err = makeTransaction(t)
	require.NoError(t, err)
	txn.Out[0].Coins = math.MaxUint64 - 3e6
	err = txn.UpdateHeader()
	require.NoError(t, err)
	uiTxn = makeUninjectedTransaction(t, &txn, 0)
	testutil.RequireError(t, uiTxn.VerifySigned(), "Output coins overflow")

	// Output coins are not multiples of 1e6 (valid, decimal restriction is not enforced here)
	txn, err = makeTransaction(t)
	require.NoError(t, err)
	txn.Out[0].Coins += 10
	err = txn.UpdateHeader()
	require.NoError(t, err)
	txn.Sigs = nil
	txn.SignInputs([]cipher.SecKey{genSecret})
	require.NotEqual(t, txn.Out[0].Coins%1e6, uint64(0))
	uiTxn = makeUninjectedTransaction(t, &txn, 0)
	require.NoError(t, uiTxn.VerifySigned())

	// Valid
	txn, err = makeTransaction(t)
	require.NoError(t, err)
	txn.Out[0].Coins = 10e6
	txn.Out[1].Coins = 1e6
	err = txn.UpdateHeader()
	require.NoError(t, err)
	uiTxn = makeUninjectedTransaction(t, &txn, 0)
	require.NoError(t, uiTxn.VerifySigned())
}

func TestUninjectedTransactionVerifyUnsigned(t *testing.T) {
	txn, _, _, err := makeTransactionMultipleInputs(t, 2)
	require.NoError(t, err)
	uiTxn := makeUninjectedTransaction(t, &txn, 0)
	err = uiTxn.VerifyUnsigned()
	testutil.RequireError(t, err, "Unsigned transaction must contain a null signature")

	// Invalid signature, not empty
	// A stable invalid signature must be used because random signatures could appear valid
	// Note: Transaction.Verify() only checks that the signature is a minimally valid signature
	badSig := "9a0f86874a4d9541f58a1de4db1c1b58765a868dc6f027445d0a2a8a7bddd1c45ea559fcd7bef45e1b76ccdaf8e50bbebd952acbbea87d1cb3f7a964bc89bf1ed5"
	txn, _, _, err = makeTransactionMultipleInputs(t, 2)
	require.NoError(t, err)
	txn.Sigs[0] = cipher.Sig{}
	txn.Sigs[1] = cipher.MustSigFromHex(badSig)
	uiTxn = makeUninjectedTransaction(t, &txn, 0)
	testutil.RequireError(t, uiTxn.VerifyUnsigned(), "Failed to recover pubkey from signature")

	txn.Sigs = nil
	uiTxn = makeUninjectedTransaction(t, &txn, 0)
	err = uiTxn.VerifyUnsigned()
	testutil.RequireError(t, err, "Invalid number of signatures")

	// Transaction is unsigned if at least 1 signature is null
	txn, _, _, err = makeTransactionMultipleInputs(t, 3)
	require.NoError(t, err)
	require.True(t, len(txn.Sigs) > 1)
	txn.Sigs[0] = cipher.Sig{}
	uiTxn = makeUninjectedTransaction(t, &txn, 0)
	err = uiTxn.VerifyUnsigned()
	require.NoError(t, err)

	// Transaction is unsigned if all signatures are null
	for i := range txn.Sigs {
		txn.Sigs[i] = cipher.Sig{}
	}
	uiTxn = makeUninjectedTransaction(t, &txn, 0)
	err = uiTxn.VerifyUnsigned()
	require.NoError(t, err)
}

func TestTransactionSignInput(t *testing.T) {
	TransactionSignInputTestImpl(t, nil)
}

func TestTransactionSignInputs(t *testing.T) {
	// Build transaction step by step
	txn := &coin.Transaction{}
	ux, kd, err := makeUxOutWithSecret(t)
	require.NoError(t, err)
	err = txn.PushInput(ux.Hash())
	require.NoError(t, err)
	wallets := makeLocalWalletsFromKeyData(t, []KeyData{*kd})
	wallet := wallets[0]
	seed, seckeys, err2 := cipher.GenerateDeterministicKeyPairsSeed([]byte(kd.Mnemonic), kd.AddressIndex+1)
	require.NoError(t, err2)
	require.Equal(t, kd.SecKey, seckeys[kd.AddressIndex])
	p2, _, err3 := cipher.GenerateDeterministicKeyPair(seed)
	require.NoError(t, err3)
	wallet.GenAddresses(core.AccountAddress, uint32(kd.AddressIndex), 2, nil)
	ux2 := coin.UxOut{
		Head: coin.UxHead{
			Time:  100,
			BkSeq: 2,
		},
		Body: coin.UxBody{
			SrcTransaction: testutil.RandSHA256(t),
			Address:        cipher.AddressFromPubKey(p2),
			Coins:          1e6,
			Hours:          100,
		},
	}
	err = txn.PushInput(ux2.Hash())
	require.NoError(t, err)
	err = txn.PushOutput(makeAddress(), 40, 80)
	require.NoError(t, err)
	require.Equal(t, len(txn.Sigs), 0)
	err = txn.UpdateHeader()
	require.NoError(t, err)
	uiTxn := makeUninjectedTransaction(t, txn, 0)
	isFullySigned, err := uiTxn.IsFullySigned()
	require.NoError(t, err)
	require.False(t, isFullySigned)

	// Mock Skycoin API calls
	mockSkyApiUxOut(global_mock, ux)
	mockSkyApiUxOut(global_mock, ux2)

	// Valid signing
	h := txn.HashInner()
	signedCoreTxn, err := wallet.Sign(uiTxn, nil, util.EmptyPassword, []string{"#0", "#1"})
	require.NoError(t, err)
	signedTxn, isUninjected := signedCoreTxn.(*SkycoinUninjectedTransaction)
	require.True(t, isUninjected)
	isFullySigned, err = signedTxn.IsFullySigned()
	require.NoError(t, err)
	require.True(t, isFullySigned)
	require.Equal(t, len(signedTxn.txn.Sigs), 2)
	h2 := signedTxn.txn.HashInner()
	require.Equal(t, h2, h)
	p := kd.PubKey
	a := cipher.AddressFromPubKey(p)
	a2 := cipher.AddressFromPubKey(p2)
	require.NoError(t, cipher.VerifyAddressSignedHash(a, signedTxn.txn.Sigs[0], cipher.AddSHA256(h, signedTxn.txn.In[0])))
	require.NoError(t, cipher.VerifyAddressSignedHash(a2, signedTxn.txn.Sigs[1], cipher.AddSHA256(h, signedTxn.txn.In[1])))
	require.Error(t, cipher.VerifyAddressSignedHash(a, signedTxn.txn.Sigs[1], cipher.AddSHA256(h, signedTxn.txn.In[0])))
	require.Error(t, cipher.VerifyAddressSignedHash(a2, signedTxn.txn.Sigs[0], cipher.AddSHA256(h, signedTxn.txn.In[1])))
}

func makeSkycoinBlockchain() core.BlockchainTransactionAPI {
	return NewSkycoinBlockchain(0)
}

func makeSkycoinSignService() core.BlockchainSignService {
	return &SkycoinSignService{}
}

func TestLocalWalletTransfer(t *testing.T) {
	CleanGlobalMock()
	destinationAddress := testutil.MakeAddress()
	sky := 500
	wlt := makeLocalWallet(t)

	addr, err := NewSkycoinAddress(destinationAddress.String())
	require.NoError(t, err)
	loadedAddrs, err := wlt.GetLoadedAddresses()
	require.NoError(t, err)
	addrs := make([]string, 0)
	for loadedAddrs.Next() {
		addrs = append(addrs, loadedAddrs.Value().String())
	}

	opt := NewTransferOptions()
	opt.SetValue("BurnFactor", "0.5")
	opt.SetValue("CoinHoursSelectionType", "auto")

	req := api.CreateTransactionRequest{
		IgnoreUnconfirmed: false,
		HoursSelection: api.HoursSelection{
			Type:        "auto",
			Mode:        "share",
			ShareFactor: "0.5",
		},
		To: []api.Receiver{
			api.Receiver{
				Address: destinationAddress.String(),
				Coins:   strconv.Itoa(sky),
			},
		},
		Addresses: addrs,
	}

	hash := testutil.RandSHA256(t)
	txn := coin.Transaction{
		Length:    100,
		Type:      0,
		InnerHash: hash,
	}
	crtTxn, err := api.NewCreateTransactionResponse(&txn, nil)
	require.NoError(t, err)
	crtTxn.Transaction.Fee = "500"

	mockSkyApiCreateTransaction(global_mock, &req, crtTxn)

	quot, err := util.AltcoinQuotient(params.SkycoinTicker)
	require.NoError(t, err)

	destination := &SkycoinTransactionOutput{
		skyOut: readable.TransactionOutput{
			Address: addr.String(),
			Coins:   util.FormatCoins(uint64(sky*1e6), quot),
		}}

	ret, err := wlt.Transfer(destination, opt)
	require.NoError(t, err)
	require.NotNil(t, ret)
	val, err := ret.ComputeFee(CoinHour)
	require.NoError(t, err)
	require.Equal(t, uint64(sky), val)
	require.Equal(t, crtTxn.Transaction.TxID, ret.GetId())
}

func TestLocalWalletSendFromAddress(t *testing.T) {
	CleanGlobalMock()
	startAddress := testutil.MakeAddress()
	destinationAddress := testutil.MakeAddress()
	changeAddress := (testutil.MakeAddress()).String()
	sky := 500
	wlt := makeLocalWallet(t)

	toAddr := &SkycoinTransactionOutput{
		skyOut: readable.TransactionOutput{
			Address: destinationAddress.String(),
			Coins:   strconv.Itoa(sky),
			Hours:   uint64(250),
		},
	}
	fromAddr, err := NewSkycoinAddress(startAddress.String())
	require.NoError(t, err)
	chgAddr, err := NewSkycoinAddress(changeAddress)
	require.NoError(t, err)

	opt1 := NewTransferOptions()
	opt1.SetValue("BurnFactor", "0.5")
	opt1.SetValue("CoinHoursSelectionType", "auto")

	req1 := api.CreateTransactionRequest{
		IgnoreUnconfirmed: false,
		HoursSelection: api.HoursSelection{
			Type:        "auto",
			Mode:        "share",
			ShareFactor: "0.5",
		},
		ChangeAddress: &changeAddress,
		To: []api.Receiver{
			api.Receiver{
				Address: destinationAddress.String(),
				Coins:   strconv.Itoa(sky),
			},
		},
		Addresses: []string{startAddress.String()},
	}

	opt2 := NewTransferOptions()
	opt2.SetValue("BurnFactor", "0.5")
	opt2.SetValue("CoinHoursSelectionType", "manual")

	req2 := api.CreateTransactionRequest{
		IgnoreUnconfirmed: false,
		HoursSelection: api.HoursSelection{
			Type: "manual",
		},
		ChangeAddress: &changeAddress,
		To: []api.Receiver{
			api.Receiver{
				Address: destinationAddress.String(),
				Coins:   strconv.Itoa(sky),
				Hours:   "250",
			},
		},
		Addresses: []string{startAddress.String()},
	}

	hash := testutil.RandSHA256(t)
	txn := coin.Transaction{
		Length:    100,
		Type:      0,
		InnerHash: hash,
	}
	crtTxn, err := api.NewCreateTransactionResponse(&txn, nil)
	require.NoError(t, err)
	crtTxn.Transaction.Fee = strconv.Itoa(sky)

	mockSkyApiCreateTransaction(global_mock, &req1, crtTxn)
	mockSkyApiCreateTransaction(global_mock, &req2, crtTxn)

	// Testing HoursSelection to auto
	ret, err := wlt.SendFromAddress([]core.Address{&fromAddr}, []core.TransactionOutput{toAddr}, &chgAddr, opt1)
	require.NoError(t, err)
	require.NotNil(t, ret)
	val, err := ret.ComputeFee(CoinHour)
	require.NoError(t, err)
	require.Equal(t, util.FormatCoins(uint64(sky), 10), util.FormatCoins(uint64(val), 10))
	require.Equal(t, crtTxn.Transaction.TxID, ret.GetId())

	// Testing HoursSelection to manual
	ret, err = wlt.SendFromAddress([]core.Address{&fromAddr}, []core.TransactionOutput{toAddr}, &chgAddr, opt2)
	require.NoError(t, err)
	require.NotNil(t, ret)
	val, err = ret.ComputeFee(CoinHour)
	require.NoError(t, err)
	require.Equal(t, util.FormatCoins(uint64(sky), 10), util.FormatCoins(uint64(val), 10))
	require.Equal(t, crtTxn.Transaction.TxID, ret.GetId())
}

func TestLocalWalletSpend(t *testing.T) {
	CleanGlobalMock()
	destinationAddress := testutil.MakeAddress()
	changeAddress := (testutil.MakeAddress()).String()
	sky := 500
	wlt := makeLocalWallet(t)

	toAddr := &SkycoinTransactionOutput{
		skyOut: readable.TransactionOutput{
			Address: destinationAddress.String(),
			Coins:   strconv.Itoa(sky),
			Hours:   uint64(250),
		},
	}
	chgAddr, err := NewSkycoinAddress(changeAddress)
	require.NoError(t, err)
	opt := NewTransferOptions()
	opt.SetValue("BurnFactor", "0.5")
	opt.SetValue("CoinHoursSelectionType", "auto")

	req := api.CreateTransactionRequest{
		IgnoreUnconfirmed: false,
		HoursSelection: api.HoursSelection{
			Type:        "auto",
			Mode:        "share",
			ShareFactor: "0.5",
		},
		ChangeAddress: &changeAddress,
		To: []api.Receiver{
			api.Receiver{
				Address: destinationAddress.String(),
				Coins:   strconv.Itoa(sky),
			},
		},
	}

	hash := testutil.RandSHA256(t)
	txn := coin.Transaction{
		Length:    100,
		Type:      0,
		InnerHash: hash,
	}
	crtTxn, err := api.NewCreateTransactionResponse(&txn, nil)
	require.NoError(t, err)
	crtTxn.Transaction.Fee = "500"

	mockSkyApiCreateTransaction(global_mock, &req, crtTxn)

	ret, err := wlt.Spend(nil, []core.TransactionOutput{toAddr}, &chgAddr, opt)
	require.NoError(t, err)
	require.NotNil(t, ret)
	val, err := ret.ComputeFee(CoinHour)
	require.NoError(t, err)
	require.Equal(t, uint64(sky), val)
	require.Equal(t, crtTxn.Transaction.TxID, ret.GetId())
}

func TestLocalWalletSignSkycoinTxn(t *testing.T) {
	CleanGlobalMock()

	//Test skycoinCreatedTransaction
	txn, keyData, uxs, err := makeTransactionMultipleInputs(t, 1)
	require.Nil(t, err)
	require.Equal(t, txn.In[0], uxs[0].Hash())
	vins := make([]visor.TransactionInput, 0)
	for _, un := range uxs {
		vin, err := visor.NewTransactionInput(un, uint64(time.Now().Unix()))
		require.Nil(t, err)
		vins = append(vins, vin)
	}
	txn.Sigs = []cipher.Sig{}
	crtTxn, err := api.NewCreatedTransaction(&txn, vins)
	require.Nil(t, err)
	require.NotNil(t, crtTxn)

	wlts := makeLocalWalletsFromKeyData(t, keyData)
	wlt := wlts[0]
	pwd := func(string, core.KeyValueStore) (string, error) {
		return "", nil
	}

	skyTxn := NewSkycoinCreatedTransaction(*crtTxn)
	sig, err := util.LookupSignServiceForWallet(wlt, core.UID(""))
	require.Nil(t, err)
	signed, err := wlt.Sign(skyTxn, sig, pwd, nil)
	require.Nil(t, err)

	ok, err := signed.IsFullySigned()
	require.Nil(t, err)
	require.True(t, ok)

	//Test that calculated hours were calculated ok
	txn.Out[0].Hours = 1000
	err = txn.UpdateHeader()
	require.Nil(t, err)
	crtTxn, err = api.NewCreatedTransaction(&txn, vins)
	require.Nil(t, err)
	require.NotNil(t, crtTxn)
	skyTxn = NewSkycoinCreatedTransaction(*crtTxn)
	sig, err = util.LookupSignServiceForWallet(wlt, core.UID(""))
	require.Nil(t, err)
	signed, err = wlt.Sign(skyTxn, sig, pwd, nil)
	require.Nil(t, err)

	ok, err = signed.IsFullySigned()
	require.Nil(t, err)
	require.True(t, ok)

}

func TestSkycoinWalletTypes(t *testing.T) {
	var wltSet core.WalletSet = &SkycoinRemoteWallet{}
	require.Equal(t, wallet.WalletTypeBip44, wltSet.DefaultWalletType())
	require.Equal(t, []string{wallet.WalletTypeDeterministic, wallet.WalletTypeBip44}, wltSet.SupportedWalletTypes())

	wltSet = &SkycoinLocalWallet{}
	require.Equal(t, wallet.WalletTypeBip44, wltSet.DefaultWalletType())
	require.Equal(t, []string{wallet.WalletTypeDeterministic, wallet.WalletTypeBip44}, wltSet.SupportedWalletTypes())
}

func TestSkycoinBlockchainSendFromAddress(t *testing.T) {
	CleanGlobalMock()

	startAddress1 := testutil.MakeAddress()
	startAddress2 := testutil.MakeAddress()

	destinationAddress := testutil.MakeAddress()
	changeAddress := testutil.MakeAddress()
	sky := 500
	hash := testutil.RandSHA256(t)

	toAddr := &SkycoinTransactionOutput{
		skyOut: readable.TransactionOutput{
			Address: destinationAddress.String(),
			Coins:   strconv.Itoa(sky),
			Hours:   uint64(250),
		},
	}
	fromAddr := []*SkycoinAddress{
		{
			address: startAddress1,
		},
		{
			address: startAddress2,
		},
	}
	chgAddr := &SkycoinAddress{
		address: changeAddress,
	}

	opt1 := NewTransferOptions()
	opt1.SetValue("BurnFactor", "0.5")
	opt1.SetValue("CoinHoursSelectionType", "auto")

	changeAddrString := changeAddress.String()
	req1 := api.CreateTransactionRequest{
		IgnoreUnconfirmed: false,
		HoursSelection: api.HoursSelection{
			Type:        "auto",
			Mode:        "share",
			ShareFactor: "0.5",
		},
		ChangeAddress: &changeAddrString,
		To: []api.Receiver{
			{
				Address: destinationAddress.String(),
				Coins:   strconv.Itoa(sky),
			},
		},
		Addresses: []string{startAddress1.String(), startAddress2.String()},
	}

	opt2 := NewTransferOptions()
	opt2.SetValue("BurnFactor", "0.5")
	opt2.SetValue("CoinHoursSelectionType", "manual")

	req2 := api.CreateTransactionRequest{
		IgnoreUnconfirmed: false,
		HoursSelection: api.HoursSelection{
			Type: "manual",
		},
		ChangeAddress: &changeAddrString,
		To: []api.Receiver{
			{
				Address: destinationAddress.String(),
				Coins:   strconv.Itoa(sky),
				Hours:   "250",
			},
		},
		Addresses: []string{startAddress1.String(), startAddress2.String()},
	}

	txn := &coin.Transaction{
		Length:    100,
		Type:      0,
		InnerHash: hash,
	}
	ctxnR, err := api.NewCreateTransactionResponse(txn, nil)
	ctxnR.Transaction.Fee = strconv.Itoa(sky)
	require.NoError(t, err)

	mockSkyApiCreateTransaction(global_mock, &req1, ctxnR)
	mockSkyApiCreateTransaction(global_mock, &req2, ctxnR)

	bc := makeSkycoinBlockchain()
	wlt := &LocalWallet{}

	// Testing Hours selection to auto
	from := []core.WalletAddress{makeSimpleWalletAddress(wlt, fromAddr[0]), makeSimpleWalletAddress(wlt, fromAddr[1])}
	to := []core.TransactionOutput{toAddr}
	txnResult, err := bc.SendFromAddress(from, to, chgAddr, opt1)
	require.NoError(t, err)
	require.NotNil(t, txnResult)
	val, err := txnResult.ComputeFee(params.CoinHoursTicker)
	require.NoError(t, err)
	require.Equal(t, util.FormatCoins(uint64(sky), 10), util.FormatCoins(uint64(val), 10))
	require.Equal(t, ctxnR.Transaction.TxID, txnResult.GetId())

	// Testing Hours selection to manual
	from = []core.WalletAddress{makeSimpleWalletAddress(wlt, fromAddr[0]), makeSimpleWalletAddress(wlt, fromAddr[1])}
	to = []core.TransactionOutput{toAddr}
	txnResult, err = bc.SendFromAddress(from, to, chgAddr, opt2)
	require.NoError(t, err)
	require.NotNil(t, txnResult)
	val, err = txnResult.ComputeFee(params.CoinHoursTicker)
	require.NoError(t, err)
	require.Equal(t, util.FormatCoins(uint64(sky), 10), util.FormatCoins(uint64(val), 10))
	require.Equal(t, ctxnR.Transaction.TxID, txnResult.GetId())

}

func TestSkycoinBlockchainSpend(t *testing.T) {
	CleanGlobalMock()

	hash := testutil.RandSHA256(t)
	sky := 500
	// chgAddr :=
	changeAddr := testutil.MakeAddress()
	chgAddr := &SkycoinAddress{
		address:     changeAddr,
		poolSection: "",
	}
	destinationAddress := testutil.MakeAddress()

	toAddr := &SkycoinTransactionOutput{
		skyOut: readable.TransactionOutput{
			Address: destinationAddress.String(),
			Coins:   strconv.Itoa(sky),
			Hours:   uint64(250),
		},
	}

	uxOuts := make([]coin.UxOut, 2)
	for i := 0; i < 2; i++ {
		ux, _, err := makeUxOutWithSecret(t)
		require.NoError(t, err)
		uxOuts[i] = ux
	}

	skyOuts := make([]core.TransactionOutput, len(uxOuts))
	for i := 0; i < len(uxOuts); i++ {
		ux := uxOuts[i]
		quot, err := util.AltcoinQuotient(params.SkycoinTicker)
		require.NoError(t, err)
		sky := util.FormatCoins(ux.Body.Coins, quot)
		skOut := SkycoinTransactionOutput{
			spent: false,
			skyOut: readable.TransactionOutput{
				Address: ux.Body.Address.String(),
				Hash:    ux.Body.Hash().String(),
				Coins:   sky,
				Hours:   ux.Body.Hours,
			},
		}
		skyOuts[i] = &skOut
	}

	wltOuts := make([]core.WalletOutput, len(uxOuts))
	for i := 0; i < len(uxOuts); i++ {
		wltOuts[i] = makeSimpleWalletOutput(nil, skyOuts[i])
	}

	uxOutsStr := make([]string, len(uxOuts))
	for i := 0; i < len(uxOuts); i++ {
		uxOutsStr[i] = uxOuts[i].Hash().String()
	}

	opt1 := NewTransferOptions()
	opt1.SetValue("BurnFactor", "0.5")
	opt1.SetValue("CoinHoursSelectionType", "auto")
	changeAddrString := changeAddr.String()
	req1 := api.CreateTransactionRequest{
		UxOuts:            uxOutsStr,
		IgnoreUnconfirmed: false,
		To: []api.Receiver{
			api.Receiver{
				Address: destinationAddress.String(),
				Coins:   strconv.Itoa(sky),
			},
		},
		HoursSelection: api.HoursSelection{
			Type:        "auto",
			Mode:        "share",
			ShareFactor: "0.5",
		},
		ChangeAddress: &changeAddrString,
	}

	opt2 := NewTransferOptions()
	opt2.SetValue("BurnFactor", "0.5")
	opt2.SetValue("CoinHoursSelectionType", "manual")

	req2 := api.CreateTransactionRequest{
		UxOuts:            uxOutsStr,
		IgnoreUnconfirmed: false,
		HoursSelection: api.HoursSelection{
			Type: "manual",
		},
		ChangeAddress: &changeAddrString,
		To: []api.Receiver{
			api.Receiver{
				Address: destinationAddress.String(),
				Coins:   strconv.Itoa(sky),
				Hours:   "250",
			},
		},
	}

	txn := coin.Transaction{
		InnerHash: hash,
		Type:      0,
		Length:    100,
	}

	crtTxn, err := api.NewCreateTransactionResponse(&txn, nil)
	require.NoError(t, err)
	crtTxn.Transaction.Fee = strconv.Itoa(sky)

	mockSkyApiCreateTransaction(global_mock, &req1, crtTxn)
	mockSkyApiCreateTransaction(global_mock, &req2, crtTxn)

	bc := makeSkycoinBlockchain()

	to := []core.TransactionOutput{toAddr}
	// Testing Hours selection auto
	txnR, err := bc.Spend(wltOuts, to, chgAddr, opt1)
	require.NoError(t, err)
	require.NotNil(t, txnR)
	require.Equal(t, txnR.GetId(), crtTxn.Transaction.TxID)
	val, err := txnR.ComputeFee(params.CoinHoursTicker)
	require.NoError(t, err)
	require.Equal(t, util.FormatCoins(uint64(sky), 10), util.FormatCoins(uint64(val), 10))

	// Testing Hours selection manual
	txnR2, err := bc.Spend(wltOuts, to, chgAddr, opt2)
	require.NoError(t, err)
	require.NotNil(t, txnR2)
	require.Equal(t, txnR2.GetId(), crtTxn.Transaction.TxID)
	val2, err := txnR2.ComputeFee(params.CoinHoursTicker)
	require.NoError(t, err)
	require.Equal(t, util.FormatCoins(uint64(sky), 10), util.FormatCoins(uint64(val2), 10))
}

func TestSkycoinSignServiceSign(t *testing.T) {
	CleanGlobalMock()

	txn, keyData, uxOuts := makeTransactionFromMultipleWallets(t, 3)
	for _, ux := range uxOuts {
		mockSkyApiUxOut(global_mock, ux)
	}

	ins := make([]visor.TransactionInput, 0)
	for _, out := range uxOuts {
		in, err := visor.NewTransactionInput(out, out.Head.Time)
		require.NoError(t, err)

		ins = append(ins, in)
	}

	pwdReader := func(_ string, _ core.KeyValueStore) (string, error) {
		return "", nil
	}

	signer := makeSkycoinSignService()
	wallets := makeLocalWalletsFromKeyData(t, keyData)

	require.NotEqual(t, wallets[0], wallets[1])

	isds := make([]core.InputSignDescriptor, 0)
	for i, wlt := range wallets {
		descriptor := core.InputSignDescriptor{
			InputIndex: fmt.Sprintf("#%d", i),
			SignerID:   "", // Use wallet
			Wallet:     wlt,
		}
		isds = append(isds, descriptor)

	}

	// SkycoinCreatedTransaction
	sigs := txn.Sigs
	txn.Sigs = []cipher.Sig{}
	apiCreTxn, err := api.NewCreatedTransaction(&txn, ins)
	txn.Sigs = sigs
	apiCreTxn.Sigs = make([]string, 0)
	require.NoError(t, err)
	require.NotNil(t, apiCreTxn)
	require.Equal(t, apiCreTxn.InnerHash, txn.HashInner().Hex())
	skyCreTxn := NewSkycoinCreatedTransaction(*apiCreTxn)
	signedTxn, err := signer.Sign(skyCreTxn, isds, pwdReader)
	require.NoError(t, err)
	require.NotNil(t, signedTxn)
	err = signedTxn.VerifySigned()
	require.NoError(t, err)

	// SkycoinUninjectedTransaction
	txn.Sigs = []cipher.Sig{}
	skyUninTxn := SkycoinUninjectedTransaction{
		txn: &txn,
		fee: 300,
	}

	signedTxn = nil

	signedTxn, err = signer.Sign(&skyUninTxn, isds, pwdReader)
	require.NoError(t, err)
	require.NotNil(t, signedTxn)

	signed, err := signedTxn.IsFullySigned()
	require.NoError(t, err)
	require.Equal(t, true, signed)

}

func TestWalletDirectoryGetStorage(t *testing.T) {
	dir := "wallet-dir"
	tests := []struct {
		wlt  *WalletDirectory
		want *SkycoinLocalWallet
	}{
		{wlt: &WalletDirectory{WalletDir: dir}, want: &SkycoinLocalWallet{walletDir: dir}},
		{wlt: &WalletDirectory{wltService: new(SkycoinLocalWallet)}, want: new(SkycoinLocalWallet)},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("GetStorage_%d", i), func(t *testing.T) {
			storage := tt.wlt.GetStorage()
			require.Equal(t, tt.want, storage)
		})
	}
}

func TestSkycoinLocalWalletListWallets(t *testing.T) {
	tests := []struct {
		dir   string
		valid bool
		want  []string
	}{
		{
			dir:   "testdata",
			valid: true,
			want:  []string{"testWallet", "encryptedWallet"},
		},
		{
			dir:   "no-dir",
			valid: false,
			want:  make([]string, 0),
		},
		{
			dir:   "testdata/invalid/wallets",
			valid: false,
			want:  make([]string, 0),
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("ListWalletsOn -> %s", tt.dir), func(t *testing.T) {
			slw := &SkycoinLocalWallet{walletDir: tt.dir}
			it := slw.ListWallets()
			labels := make([]string, 0)
			if tt.valid {
				require.NotNil(t, it)
				for it.Next() {
					wlt := it.Value()
					labels = append(labels, wlt.GetLabel())
				}
			} else {
				require.Nil(t, it)
			}
			sort.Strings(labels)
			sort.Strings(tt.want)
			require.Equal(t, tt.want, labels)
		})
	}
}

func TestSkycoinLocalIsEncrypted(t *testing.T) {
	slw := &SkycoinLocalWallet{walletDir: "testdata"}
	tests := []struct {
		srv   *SkycoinLocalWallet
		name  string
		valid bool
		want  bool
	}{
		{srv: slw, valid: true, want: false, name: "test.wlt"},
		{srv: slw, valid: true, want: true, name: "encrypted.wlt"},
		{srv: slw, valid: false, want: false, name: "unknown.wlt"},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("Wallet%d -> %s", i, tt.name), func(t *testing.T) {
			encrypted, err := tt.srv.IsEncrypted(tt.name)
			require.Equal(t, tt.want, encrypted)
			if tt.valid {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
			}
		})
	}
}

func TestSkycoinLocalWalletEncrypt(t *testing.T) {
	slw := &SkycoinLocalWallet{walletDir: "testdata"}
	pwd := func(s string, store core.KeyValueStore) (string, error) {
		return "test-password", nil
	}
	emptyPwd := func(s string, store core.KeyValueStore) (string, error) {
		return "", nil
	}
	tests := []struct {
		srv   *SkycoinLocalWallet
		pwd   core.PasswordReader
		name  string
		valid bool
	}{
		{srv: slw, pwd: pwd, valid: true, name: "test.wlt"},
		{srv: slw, pwd: pwd, valid: true, name: "encrypted.wlt"},
		{srv: slw, pwd: pwd, valid: false, name: "unknown.wlt"},
		{srv: slw, pwd: emptyPwd, valid: false, name: "test.wlt"},
		{srv: slw, pwd: emptyPwd, valid: true, name: "encrypted.wlt"},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("Wallet%d -> %s", i, tt.name), func(t *testing.T) {
			wlt, err := wallet.Load(filepath.Join(tt.srv.walletDir, tt.name))
			clean := err == nil

			tt.srv.Encrypt(tt.name, tt.pwd)
			encrypted, err := tt.srv.IsEncrypted(tt.name)
			if clean {
				require.NoError(t, err)
				err = wallet.Save(wlt, tt.srv.walletDir)
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			require.Equal(t, tt.valid, encrypted)
		})
	}
}

func TestSkycoinLocalWalletDecrypt(t *testing.T) {
	slw := &SkycoinLocalWallet{walletDir: "testdata"}
	pwd := func(s string, store core.KeyValueStore) (string, error) {
		return "test-password", nil
	}
	emptyPwd := func(s string, store core.KeyValueStore) (string, error) {
		return "", nil
	}
	tests := []struct {
		srv   *SkycoinLocalWallet
		pwd   core.PasswordReader
		name  string
		valid bool
	}{
		{srv: slw, pwd: pwd, valid: false, name: "test.wlt"},
		{srv: slw, pwd: pwd, valid: false, name: "encrypted.wlt"},
		{srv: slw, pwd: pwd, valid: false, name: "unknown.wlt"},
		{srv: slw, pwd: emptyPwd, valid: false, name: "test.wlt"},
		{srv: slw, pwd: emptyPwd, valid: true, name: "encrypted.wlt"},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("Wallet%d -> %s", i, tt.name), func(t *testing.T) {
			wlt, err := wallet.Load(filepath.Join(tt.srv.walletDir, tt.name))
			clean := err == nil

			tt.srv.Decrypt(tt.name, tt.pwd)
			encrypted, err := tt.srv.IsEncrypted(tt.name)
			if clean {
				require.NoError(t, err)
				err = wallet.Save(wlt, tt.srv.walletDir)
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			require.Equal(t, tt.valid, encrypted)
		})
	}
}

func TestLocalWalletSetLabel(t *testing.T) {
	newLabel := "custom-label"
	tests := []struct {
		name  string
		label string
		valid bool
	}{
		{label: "testWallet", name: "test.wlt", valid: true},
		{label: "encryptedWallet", name: "encrypted.wlt", valid: true},
		{name: "unknown.wlt", valid: false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("CangeLabelOf -> %s", tt.name), func(t *testing.T) {
			lw := &LocalWallet{WalletDir: "testdata", Id: tt.name}
			lw.SetLabel(newLabel)
			if tt.valid {
				label := lw.GetLabel()
				lw.SetLabel(tt.label)
				require.Equal(t, newLabel, label)
			}
		})
	}
}

func TestWalletsReadyForTxn(t *testing.T) {
	type WalleSigner struct {
		mocks.Wallet
		mocks.TxnSigner
	}
	emptyWlt := new(LocalWallet)
	mockWlt := new(WalleSigner)
	mockWlt.TxnSigner.On(
		"ReadyForTxn",
		mock.AnythingOfType("*skycoin.LocalWallet"),
		mock.AnythingOfType("*mocks.Transaction"),
	).Return(
		func(w core.Wallet, txn core.Transaction) bool {
			ok, err := checkTxnSupported(mockWlt, w, txn)
			require.NoError(t, err)
			return ok
		},
		nil,
	)

	tests := []struct {
		signer core.TxnSigner
		wlt2   core.Wallet
		txn    core.Transaction
		valid  bool
		want   bool
	}{
		{
			valid:  true,
			want:   false,
			signer: mockWlt,
			wlt2:   new(LocalWallet),
			txn:    new(mocks.Transaction),
		},
		{
			valid:  true,
			want:   false,
			signer: new(LocalWallet),
			wlt2:   mockWlt,
			txn:    new(mocks.Transaction),
		},
		{
			valid:  true,
			want:   false,
			signer: &LocalWallet{Type: "custom-type"},
			wlt2:   new(LocalWallet),
			txn:    new(mocks.Transaction),
		},
		{
			valid:  false,
			want:   false,
			signer: new(LocalWallet),
			wlt2:   new(LocalWallet),
			txn:    new(mocks.Transaction),
		},
		{
			valid:  true,
			want:   false,
			signer: emptyWlt,
			wlt2:   emptyWlt,
			txn:    new(mocks.Transaction),
		},
		{
			valid:  false,
			want:   false,
			signer: &LocalWallet{WalletDir: "testdata", Id: "test.wlt"},
			wlt2:   new(LocalWallet),
			txn:    new(mocks.Transaction),
		},
		{
			valid:  true,
			want:   true,
			signer: &LocalWallet{WalletDir: "testdata", Id: "test.wlt"},
			wlt2:   &LocalWallet{WalletDir: "testdata", Id: "test.wlt"},
			txn:    new(SkycoinTransaction),
		},
		{
			valid:  true,
			want:   false,
			signer: &LocalWallet{WalletDir: "testdata", Id: "test.wlt"},
			wlt2:   &LocalWallet{WalletDir: "testdata", Id: "test.wlt"},
			txn:    new(mocks.Transaction),
		},
		//RemoteWallet
		{
			valid:  false,
			want:   false,
			signer: new(RemoteWallet),
			wlt2:   new(RemoteWallet),
			txn:    new(mocks.Transaction),
		},
	}

	for _, tt := range tests {
		t.Run("ReadyForTxn", func(t *testing.T) {
			ready, err := tt.signer.ReadyForTxn(tt.wlt2, tt.txn)
			if tt.valid {
				require.Equal(t, tt.want, ready)
			} else {
				require.NotNil(t, err)
			}
		})
	}
}

func TestWalletFunctions(t *testing.T) {
	id := "local-id"
	wlt := &LocalWallet{Id: id}
	signerUuid, err := wlt.GetSignerUID()
	require.NoError(t, err)
	require.Equal(t, core.UID(SignerIDLocalWallet), signerUuid)
	signerDescription, err := wlt.GetSignerDescription()
	require.NoError(t, err)
	require.Equal(t, "Remote Skycoin wallet "+id, signerDescription)

	rmt := &RemoteWallet{Id: id}
	signerUuid, err = rmt.GetSignerUID()
	require.NoError(t, err)
	require.Equal(t, core.UID(SignerIDRemoteWallet), signerUuid)
	signerDescription, err = rmt.GetSignerDescription()
	require.NoError(t, err)
	require.Equal(t, "Remote Skycoin wallet "+id, signerDescription)
}

func TestWalletNodeGetWalletSet(t *testing.T) {
	sectionName := "custom-section"
	tests := []struct {
		wlt  *WalletNode
		want core.WalletSet
	}{
		{wlt: new(WalletNode), want: new(SkycoinRemoteWallet)},
		{wlt: &WalletNode{poolSection: sectionName}, want: &SkycoinRemoteWallet{poolSection: sectionName}},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("GetWalletSet%d", i), func(t *testing.T) {
			require.Equal(t, tt.want, tt.wlt.GetWalletSet())
		})
	}
}

func TestWalletNodeGetStorage(t *testing.T) {
	sectionName := "custom-section"
	tests := []struct {
		wlt  *WalletNode
		want core.WalletSet
	}{
		{wlt: new(WalletNode), want: new(SkycoinRemoteWallet)},
		{wlt: &WalletNode{poolSection: sectionName}, want: &SkycoinRemoteWallet{poolSection: sectionName}},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("GetWalletSet%d", i), func(t *testing.T) {
			require.Equal(t, tt.want, tt.wlt.GetStorage())
		})
	}
}

func TestSeedServiceGenerateMnemonic(t *testing.T) {
	srv := new(SeedService)
	tests := []struct {
		name  string
		bits  int
		valid bool
	}{
		{name: "bad-entropyBits", bits: 15, valid: false},
		{name: "128-bits", bits: 128, valid: true},
		{name: "256-bits", bits: 256, valid: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := srv.GenerateMnemonic(tt.bits)
			if tt.valid {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
			}
		})
	}
}

func TestSeedServiceVerifyMnemonic(t *testing.T) {
	srv := new(SeedService)
	mnc128, err := srv.GenerateMnemonic(128)
	require.NoError(t, err)
	mnc256, err := srv.GenerateMnemonic(256)
	require.NoError(t, err)
	tests := []struct {
		name     string
		mnemonic string
		valid    bool
	}{
		{name: "bad-mnemonic", mnemonic: "invalid-mnemonic", valid: false},
		{name: "128-bits", mnemonic: mnc128, valid: true},
		{name: "256-bits", mnemonic: mnc256, valid: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := srv.VerifyMnemonic(tt.mnemonic)
			if tt.valid {
				require.True(t, valid)
				require.Nil(t, err)
			} else {
				require.False(t, valid)
				require.NotNil(t, err)
			}
		})
	}
}

func TestErrorTickerInvalidError(t *testing.T) {
	format := " is an invalid ticker. Use " + Sky + " or " + CoinHour
	tickers := []string{"a", "b", "c"}
	for _, ticker := range tickers {
		err := errorTickerInvalid{ticker}
		require.Equal(t, ticker+format, err.Error())
	}
}

func TestNewWalletNode(t *testing.T) {
	addr := "addr"
	for i := 1; i < 4; i++ {
		wn := NewWalletNode(addr)
		require.Equal(t, fmt.Sprintf("skycoin-%d", i), wn.poolSection)
		require.Equal(t, addr, wn.NodeAddress)
	}
}

package hub

import (
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/brucexc/pray-to-earn/contract"
	"github.com/brucexc/pray-to-earn/internal/service/hub/model/errorx"
	"github.com/creasty/defaults"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type KnockRequest struct {
	Address common.Address `json:"address" validate:"required"`
	Note    string         `json:"note"`
}

type Response struct {
	Data any `json:"data"`
}

type KnockResponse struct {
	Note string `json:"note"`
}

type PeekNoteRequest struct {
	common.Address `json:"address" validate:"required"`
	TxHash         common.Hash `json:"tx_hash" validate:"required"`
}

type PeekNoteResponse struct {
	Note string `json:"note"`
}

type ReplyRequest struct {
	Address common.Address `json:"address" validate:"required"`
	Note    string         `json:"note" validate:"required"`
}

type ReplyResponse struct {
	Note string `json:"note"`
}

type Note struct {
	Time    time.Time      `json:"time"`
	Address common.Address `json:"address"`
	Note    string         `json:"note"`
}

const notesSet = "notes_set"

var zeroAddress = common.HexToAddress("0x0000000000000000000000000000000000000000")
var peekNodePrice = big.NewInt(1).Mul(big.NewInt(1e18), big.NewInt(10))

func (h *Hub) Knock(c echo.Context) error {
	var request KnockRequest

	if err := c.Bind(&request); err != nil {
		return errorx.BadParamsError(c, fmt.Errorf("bind request: %w", err))
	}

	if err := defaults.Set(&request); err != nil {
		zap.L().Error("set default values for request", zap.Error(err))

		return errorx.InternalError(c)
	}

	if err := c.Validate(&request); err != nil {
		return errorx.ValidationFailedError(c, fmt.Errorf("validation failed: %w", err))
	}

	mintTokens := big.NewInt(1e18)
	var otherNote string
	if request.Note != "" {
		mintTokens = big.NewInt(1).Mul(big.NewInt(1e18), big.NewInt(5))

		h.redisClient.SAdd(c.Request().Context(), notesSet, request.Note)
		otherNote, _ = h.redisClient.SRandMember(c.Request().Context(), notesSet).Result()
	}

	tx, err := h.prayContract.Mint(h.auth, request.Address, mintTokens)
	if err != nil {
		return errorx.InternalError(c)
	}

	receipt, err := bind.WaitMined(c.Request().Context(), h.ethereumClient, tx)
	if err != nil || receipt.Status != types.ReceiptStatusSuccessful {
		return errorx.InternalError(c)
	}

	zap.L().Info("minted tokens", zap.String("to", request.Address.Hex()), zap.Any("quantity", mintTokens),
		zap.String("tx_hash", tx.Hash().Hex()),
		zap.String("note", request.Note), zap.String("other_note", otherNote))

	return c.JSON(http.StatusOK, Response{
		Data: KnockResponse{
			Note: otherNote,
		},
	})
}

func (h *Hub) PeekNote(c echo.Context) error {
	var request PeekNoteRequest

	if err := c.Bind(&request); err != nil {
		return errorx.BadParamsError(c, fmt.Errorf("bind request: %w", err))
	}

	if err := defaults.Set(&request); err != nil {
		zap.L().Error("set default values for request", zap.Error(err))

		return errorx.InternalError(c)
	}

	if err := c.Validate(&request); err != nil {
		return errorx.ValidationFailedError(c, fmt.Errorf("validation failed: %w", err))
	}

	zap.L().Info("peek note", zap.String("tx_hash", request.TxHash.Hex()), zap.String("address", request.Address.Hex()))

	if err := h.verifyTxPayment(c, request); err != nil {
		return err
	}

	// get a random note from the notes set
	note, err := h.redisClient.SRandMember(c.Request().Context(), notesSet).Result()
	if err != nil {
		zap.L().Error("failed to get a random note", zap.Error(err))
		return errorx.InternalError(c)
	}

	return c.JSON(http.StatusOK, Response{
		Data: PeekNoteResponse{
			Note: note,
		},
	})
}

func (h *Hub) verifyTxPayment(c echo.Context, request PeekNoteRequest) error {
	receipt, err := h.ethereumClient.TransactionReceipt(c.Request().Context(), request.TxHash)
	if err != nil {
		return errorx.ValidationFailedError(c, fmt.Errorf("get transaction receipt: %w", err))
	}

	// check if the transaction is a burn transaction
	for _, log := range receipt.Logs {
		if log.Address == contract.AddressPray && log.Topics[0] == contract.TransferEventSig {
			from := common.HexToAddress(log.Topics[1].Hex())
			to := common.HexToAddress(log.Topics[2].Hex())
			amount := new(big.Int).SetBytes(log.Data)

			if from == request.Address && to == zeroAddress && amount.Cmp(peekNodePrice) == 0 {
				zap.L().Info("found payment", zap.Any("from", from), zap.Any("to", to), zap.Any("amount", amount))
				return nil
			}
		}
	}
	return errorx.BadPaymentError(c, fmt.Errorf("payment not found"))
}

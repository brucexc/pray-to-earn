package hub

import (
	"fmt"
	"math/big"
	"net/http"
	"time"

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

type Note struct {
	Time    time.Time      `json:"time"`
	Address common.Address `json:"address"`
	Note    string         `json:"note"`
}

const notesSet = "notes_set"

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

	mintTokens := big.NewInt(1)
	var otherNote string
	if request.Note != "" {
		mintTokens = big.NewInt(5)

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

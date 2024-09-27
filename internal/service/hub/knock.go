package hub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"math/big"
	"math/rand"
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

type ReplyRequest struct {
	ID      string         `json:"id" validate:"required"`
	Note    string         `json:"note" validate:"required"`
	Address common.Address `json:"address" validate:"required"`
}

type Response struct {
	Data any `json:"data"`
}

type KnockResponse struct {
	TotalTokens *big.Int `json:"total_tokens"`
	AddTokens   *big.Int `json:"add_tokens"`
	Note        *Message `json:"note"`
}

type Message struct {
	ID      string   `json:"id"`
	Content string   `json:"content"`
	Replies []string `json:"replies"`
}

type PeekNoteRequest struct {
	Address common.Address `json:"address" validate:"required"`
	TxHash  common.Hash    `json:"tx_hash" validate:"required"`
}

type PeekNoteResponse struct {
	Note *Message `json:"note"`
}

type FaucetRequest struct {
	Address common.Address `json:"address" validate:"required"`
}

type FaucetResponse struct {
	Success bool        `json:"success"`
	TxHash  common.Hash `json:"tx_hash" validate:"required"`
}

const messagesSet = "messages_set"

var zeroAddress = common.HexToAddress("0x0000000000000000000000000000000000000000")
var peekNodePrice = big.NewInt(1).Mul(big.NewInt(1e18), big.NewInt(10))
var serverAdminAddress = common.HexToAddress("0xBd7537Df4991ef4ABc245e48989C5beE6A56fC61")

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

	// rate limit
	success, err := h.redisClient.SetNX(c.Request().Context(), request.Address.String(), 1, 5*time.Second).Result()
	if err != nil || !success {
		return errorx.TooManyRequestError(c, fmt.Errorf("too many requests"))
	}

	mintTokens := big.NewInt(1e18)
	var otherNote *Message
	if request.Note != "" {
		// mint 5-10 tokens
		mintTokens = big.NewInt(1).Mul(big.NewInt(1e18), big.NewInt(int64(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(6)+5)))

		storeNote := fmt.Sprintf("%s %s: %s", time.Now().Format("2006-01-02 15:04:05"), request.Address.Hex()[:8], request.Note)
		_, _ = h.storeMessage(c.Request().Context(), storeNote)
		otherNote, _ = h.getRandomMessage(c.Request().Context())
	}

	tx, err := h.prayContract.Mint(h.auth, request.Address, mintTokens)
	if err != nil {
		return errorx.InternalError(c)
	}

	receipt, err := bind.WaitMined(c.Request().Context(), h.ethereumClient, tx)
	if err != nil || receipt.Status != types.ReceiptStatusSuccessful {
		return errorx.InternalError(c)
	}

	totalTokens, _ := h.prayContract.BalanceOf(&bind.CallOpts{}, request.Address)

	zap.L().Info("minted tokens", zap.String("to", request.Address.Hex()), zap.Any("quantity", mintTokens),
		zap.String("tx_hash", tx.Hash().Hex()),
		zap.String("note", request.Note), zap.Any("other_note", otherNote))

	return c.JSON(http.StatusOK, Response{
		Data: KnockResponse{
			TotalTokens: totalTokens,
			AddTokens:   mintTokens,
			Note:        otherNote,
		},
	})
}

func (h *Hub) Reply(c echo.Context) error {
	var request ReplyRequest

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

	storeNote := fmt.Sprintf("%s %s: %s", time.Now().Format("2006-01-02 15:04:05"), request.Address.Hex()[:8], request.Note)
	_, _ = h.addReplyToMessage(c.Request().Context(), request.ID, storeNote)

	zap.L().Info("replied to note", zap.String("id", request.ID), zap.String("note", request.Note))

	return c.JSON(http.StatusOK, Response{
		Data: "ok",
	})
}

func (h *Hub) storeMessage(ctx context.Context, content string) (*Message, error) {
	messageID := uuid.New().String()
	message := &Message{
		ID:      messageID,
		Content: content,
		Replies: []string{},
	}

	messageJSON, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}

	messageKey := fmt.Sprintf("message:%s", messageID)
	err = h.redisClient.Set(ctx, messageKey, messageJSON, 0).Err()
	if err != nil {
		return nil, err
	}

	err = h.redisClient.SAdd(ctx, "messages_set", messageID).Err()
	if err != nil {
		return nil, err
	}

	return message, nil
}

func (h *Hub) getRandomMessage(ctx context.Context) (*Message, error) {
	messageID, err := h.redisClient.SRandMember(ctx, messagesSet).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("no messgae found")
		}
		return nil, err
	}

	messageKey := fmt.Sprintf("message:%s", messageID)
	messageJSON, err := h.redisClient.Get(ctx, messageKey).Result()
	if err != nil {
		return nil, err
	}

	var message Message
	err = json.Unmarshal([]byte(messageJSON), &message)
	if err != nil {
		return nil, err
	}

	return &message, nil
}

func (h *Hub) addReplyToMessage(ctx context.Context, messageID string, reply string) (*Message, error) {
	messageKey := fmt.Sprintf("message:%s", messageID)
	messageJSON, err := h.redisClient.Get(ctx, messageKey).Result()
	if err != nil {
		return nil, err
	}

	var message Message
	err = json.Unmarshal([]byte(messageJSON), &message)
	if err != nil {
		return nil, err
	}

	message.Replies = append(message.Replies, reply)

	updatedMessageJSON, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}

	err = h.redisClient.Set(ctx, messageKey, updatedMessageJSON, 0).Err()
	if err != nil {
		return nil, err
	}

	return &message, nil
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

	otherNote, err := h.getRandomMessage(c.Request().Context())
	if err != nil {
		zap.L().Error("failed to get a random note", zap.Error(err))
		return errorx.InternalError(c)
	}

	return c.JSON(http.StatusOK, Response{
		Data: PeekNoteResponse{
			Note: otherNote,
		},
	})
}

func (h *Hub) Faucet(c echo.Context) error {
	var request FaucetRequest

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

	zap.L().Info("send 0.5 RSS3 to", zap.String("address", request.Address.Hex()))

	// send 0.5 ether to request.Address, use h.auth.Signer to sign the transaction
	nonce, _ := h.ethereumClient.NonceAt(c.Request().Context(), serverAdminAddress, nil)
	gasPrice, _ := h.ethereumClient.SuggestGasPrice(c.Request().Context())
	sendTx, err := h.auth.Signer(h.auth.From, types.NewTransaction(nonce, request.Address, big.NewInt(5e17), 21000, gasPrice, nil))
	if err != nil {
		zap.L().Error("failed to sign transaction", zap.Error(err))
		return errorx.InternalError(c)
	}

	zap.L().Info("send 0.5 rss3 to ", zap.String("address", request.Address.Hex()), zap.String("tx_hash", sendTx.Hash().Hex()))

	if err := h.ethereumClient.SendTransaction(c.Request().Context(), sendTx); err != nil {
		zap.L().Error("failed to send transaction", zap.Error(err))
		return errorx.InternalError(c)
	}

	return c.JSON(http.StatusOK, Response{
		Data: FaucetResponse{
			Success: true,
			TxHash:  sendTx.Hash(),
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

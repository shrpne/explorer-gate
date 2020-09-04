package errors

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/ptypes/struct"
	"google.golang.org/grpc/status"
	"math/big"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

func NewGateError(msg string) GateError {
	return GateError{
		ErrorString: "",
		Code:        "1",
		Message:     msg,
		Details:     nil,
	}
}

func SetErrorResponse(err error, c *gin.Context) {
	grpcErr, ok := status.FromError(err)
	if !ok {
		c.JSON(http.StatusBadRequest, NewGateError(`want error type: "GRPC Status"`))
		return
	}

	var result GateError
	msg, e := formatErrorMessage(grpcErr.Message())

	code := fmt.Sprintf("%d", int(grpcErr.Code()))

	details := make(map[string]interface{})

	for _, v := range grpcErr.Details() {
		dd, ok := v.(*structpb.Struct)
		if ok {
			for k, v := range dd.AsMap() {
				details[k] = v
				if k == "code" {
					code = v.(string)
				}
			}
		}
	}

	if e != nil {
		result = NewGateError(e.Error())
	} else {
		result = GateError{
			ErrorString: "",
			Code:        code,
			Message:     msg,
			Details:     details,
		}
	}
	c.JSON(http.StatusBadRequest, result)
}

func formatErrorMessage(errorString string) (string, error) {
	bip := big.NewFloat(0.000000000000000001)
	zero := big.NewFloat(0)

	re := regexp.MustCompile(`(?mi)(Has:|required|Wanted|Expected|maximum|spend|minimum|get) -*(\d+)`)
	matches := re.FindAllStringSubmatch(errorString, -1)

	if matches != nil {
		for _, match := range matches {
			var valueString string
			value, _, err := big.ParseFloat(match[2], 10, 0, big.ToZero)
			if err != nil {
				return "", err
			}
			value = value.Mul(value, bip)

			if value.Cmp(zero) == 0 {
				valueString = "0"
			} else {
				valueString = value.Text('f', 10)
			}

			f, err := strconv.ParseFloat(valueString, 10)
			replacer := strings.NewReplacer(match[2], strconv.FormatFloat(f, 'f', -1, 64))
			errorString = replacer.Replace(errorString)
		}
	}
	return errorString, nil
}

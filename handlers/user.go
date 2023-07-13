package handlers

import (
	authDto "accounting-project/dto/auth"
	"accounting-project/dto/otp"
	dto "accounting-project/dto/result"
	"accounting-project/models"
	jwtToken "accounting-project/pkg/jwt"
	"accounting-project/pkg/redis"
	"accounting-project/repositories"
	"context"
	"encoding/json"
	"errors"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Handler struct {
	UserRepository repositories.UserRepository
}

func HandlerUser(UserRepository repositories.UserRepository) *Handler {
	return &Handler{UserRepository: UserRepository}
}
func (h *Handler) SignUp(c echo.Context) error {
	request := new(authDto.SignUpRequest)
	err := c.Bind(request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResult{Code: http.StatusBadRequest, Message: err.Error()})
	}

	validation := validator.New()
	err = validation.Struct(request)

	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResult{Code: http.StatusBadRequest, Message: err.Error()})
	}

	//check mobileNumber not exist in db
	user := h.UserRepository.FindUserByMobileNumber(request.MobileNumber)
	if user.Id != 0 {
		return c.JSON(http.StatusBadRequest, dto.ErrorResult{Code: http.StatusBadRequest, Message: "This MobileNumber already exist"})
	}
	//code, err := sendSMS(request.MobileNumber)
	//if err != nil {
	//	return c.JSON(http.StatusServiceUnavailable, dto.ErrorResult{Code: http.StatusServiceUnavailable, Message: err.Error()})
	//}
	code := "123456"
	//add code to request model and convert to json
	request.Code = code
	redisValue, err := json.Marshal(request)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResult{Code: http.StatusInternalServerError, Message: err.Error()})

	}
	//set mobile number and code in redis
	var ctx = context.Background()
	err = redis.Rdb.Set(ctx, request.MobileNumber, string(redisValue), time.Minute+2*time.Second).Err()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResult{Code: http.StatusInternalServerError, Message: err.Error()})
	}
	return c.JSON(http.StatusOK, dto.SuccessResult{Code: http.StatusOK, Data: request.MobileNumber})
}

func (h *Handler) Verify(c echo.Context) error {
	request := new(authDto.VerifyRequest)
	err := c.Bind(request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResult{Code: http.StatusBadRequest, Message: err.Error()})
	}
	//get origin code with mobile number in redis
	ctx := context.Background()
	redisValue, err := redis.Rdb.Get(ctx, request.MobileNumber).Result()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResult{Code: http.StatusInternalServerError, Message: err.Error()})
	}
	//convert redis value to signUp value and get code
	var signUpRequest authDto.SignUpRequest
	err = json.Unmarshal([]byte(redisValue), &signUpRequest)
	//check input code with origin code
	if signUpRequest.Code != request.Code {
		return c.JSON(http.StatusBadRequest, dto.ErrorResult{Code: http.StatusBadRequest, Message: "Code not valid"})
	}
	//hashing password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(signUpRequest.Password), 10)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResult{Code: http.StatusInternalServerError, Message: err.Error()})
	}
	//Create User And save to db
	user := models.User{
		FirstName:    signUpRequest.FirstName,
		LastName:     signUpRequest.LastName,
		MobileNumber: request.MobileNumber,
		Password:     string(hashedPassword),
		IsVerify:     true,
		RegisterTime: time.Now(),
	}
	user, err = h.UserRepository.CreateUser(user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResult{Code: http.StatusInternalServerError, Message: err.Error()})
	}

	//generate token
	claims := jwt.MapClaims{}
	claims["id"] = user.Id
	claims["exp"] = time.Now().Add(time.Hour * 24 * 30).Unix() // 2 hours expired

	token, errGenerateToken := jwtToken.GenerateToken(&claims)
	if errGenerateToken != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResult{Code: http.StatusInternalServerError, Message: err.Error()})
	}
	response := authDto.SignUpResponse{
		User:  user,
		Token: token,
	}

	return c.JSON(http.StatusOK, dto.SuccessResult{Code: http.StatusOK, Data: response})

}
func (h *Handler) Login(c echo.Context) error {
	request := new(authDto.LoginRequest)
	err := c.Bind(request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResult{Code: http.StatusBadRequest, Message: err.Error()})
	}

	validation := validator.New()
	err = validation.Struct(request)

	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResult{Code: http.StatusBadRequest, Message: err.Error()})
	}
	//check user exist
	user := h.UserRepository.FindUserByMobileNumber(request.MobileNumber)
	if user.Id == 0 {
		return c.JSON(http.StatusBadRequest, dto.ErrorResult{Code: http.StatusBadRequest, Message: "This Email not exist"})
	}
	//generate token
	claims := jwt.MapClaims{}
	claims["id"] = user.Id
	claims["exp"] = time.Now().Add(time.Hour * 24 * 30).Unix() // 2 hours expired

	token, errGenerateToken := jwtToken.GenerateToken(&claims)
	if errGenerateToken != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResult{Code: http.StatusInternalServerError, Message: err.Error()})
	}

	response := authDto.SignUpResponse{
		User:  user,
		Token: token,
	}

	return c.JSON(http.StatusOK, dto.SuccessResult{Code: http.StatusOK, Data: response})

}

func (h *Handler) Forgot(c echo.Context) error {
	request := new(authDto.ForgotRequest)
	err := c.Bind(request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResult{Code: http.StatusBadRequest, Message: err.Error()})
	}

	validation := validator.New()
	err = validation.Struct(request)

	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResult{Code: http.StatusBadRequest, Message: err.Error()})
	}
	//check user exist
	user := h.UserRepository.FindUserByMobileNumber(request.Email)
	if user.Id == 0 {
		return c.JSON(http.StatusBadRequest, dto.ErrorResult{Code: http.StatusBadRequest, Message: "This Email not exist"})
	}
	password := "Aa123456789Aa"
	//hashing password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResult{Code: http.StatusInternalServerError, Message: err.Error()})
	}
	user.Password = string(hashedPassword)

	return c.JSON(http.StatusOK, dto.SuccessResult{Code: http.StatusOK, Data: ""})

}

func sendSMS(mobileNumber string) (string, error) {
	client := &http.Client{}
	url := os.Getenv("KAVENEGAR_URL")
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", errors.New("errored when create request for sms server")
	}
	// set seed
	rand.Seed(time.Now().UnixNano())
	// generate random number and print on console
	code := strconv.Itoa(rand.Intn(999999-100000) + 10000)
	q := req.URL.Query()
	q.Add("receptor", mobileNumber)
	q.Add("token", code)
	q.Add("template", "accounting")
	req.URL.RawQuery = q.Encode()
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.New("errored when sending request to the server")
	}
	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)
	var response otp.Response
	err = json.Unmarshal(respBody, &response)
	if err != nil {
		return "", errors.New("errored when converting json to response model")
	}
	if response.Return.Status == http.StatusOK {
		return code, nil
	}
	return "", errors.New(response.Entries[0].Message)
}

func (h *Handler) CheckAuth(c echo.Context) error {
	userLogin := c.Get("userLogin")
	userId := userLogin.(jwt.MapClaims)["id"].(float64)

	user := h.UserRepository.CheckAuth(int(userId))

	return c.JSON(http.StatusOK, dto.SuccessResult{Code: http.StatusOK, Data: user})
}

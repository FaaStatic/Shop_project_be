package response

import "github.com/gofiber/fiber/v3"

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

func Success(c fiber.Ctx, statusCode int, message string, data interface{}) error {
	return c.Status(statusCode).JSON(APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func Error(c fiber.Ctx, statusCode int, message string, err error) error {
	resp := APIResponse{
		Success: false,
		Message: message,
	}
	if err != nil {
		resp.Error = err.Error()
	}
	return c.Status(statusCode).JSON(resp)
}

func Paginated(c fiber.Ctx, message string, data interface{}, meta interface{}) error {
	return c.Status(200).JSON(APIResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

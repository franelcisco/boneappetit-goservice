package routers

import (
	"bone_appetit_suscription/internal/handlers"

	"github.com/gin-gonic/gin"
)

type OrderRouter struct {
	handler handlers.OrderHandler
}

func NewOrderRouter(handler handlers.OrderHandler) OrderRouter {
	return OrderRouter{handler: handler}
}

func (s *OrderRouter) SetRouter(router *gin.Engine) {
	router.POST("suscription/schedule-next-order", s.handler.ScheduleNextOrder)
	router.GET("/orders/ondemand/:order_no", s.handler.CreateOnDemandOrderHandler)
}

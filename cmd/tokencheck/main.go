// cmd/tokencheck — разовая проверка токена бота через GET /me.
// Утилита для отладки: печатает информацию о боте либо текст ошибки.
// Токен в вывод не попадает.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	maxbot "github.com/max-messenger/max-bot-api-client-go"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("ПРЕДУПРЕЖДЕНИЕ: .env не найден, беру переменные окружения")
	}

	token := os.Getenv("MAX_BOT_TOKEN")
	if token == "" {
		fmt.Println("ОТКАЗ: MAX_BOT_TOKEN пуст")
		os.Exit(1)
	}

	api, err := maxbot.New(token)
	if err != nil {
		fmt.Printf("ОТКАЗ: не удалось создать клиент: %v\n", err)
		os.Exit(1)
	}

	info, err := api.Bots.GetBot(context.Background())
	if err != nil {
		fmt.Printf("НЕ РАБОТАЕТ: MAX API отклонил запрос: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("РАБОТАЕТ: бот «%s» (ID: %d, @%s)\n", info.Name, info.UserId, info.Username)
}

package main

import (
	"fmt"
	"strings"
)

// parseArtists парсит список артистов из строки (копия из handlers)
func parseArtists(input string) []string {
	// Разделяем по запятым и очищаем от пробелов
	parts := strings.Split(input, ",")
	var artists []string
	for _, part := range parts {
		artist := strings.TrimSpace(part)
		if artist != "" {
			artists = append(artists, artist)
		}
	}
	return artists
}

func main() {
	// Тестируем парсинг артистов из примера пользователя
	testInput := "ablume, aespa, apink, artms, atheart, baby dont cry, babymonster, badvillain, bewave, bibi, billlie, blackpink, burvey, candy shop, chuu, class:y, crazangel, dreamnote, fifty fifty, fromis_9, gfriend, gyubin, h1-key, hanhee, hearts2hearts, hitgs, hyuna, i-dle, ichillin', ifeye, illit, irene, irene & seulgi, itzy, iu, ive, izna, jennie, jeon somi, kandis, kep1er, kiiikiii, kiiras, kiss of life, kwon eunbi, le sserafim, lightsum, lisa, meovv, meowmiro, minnie, misamo, miyeon, moon byul, mrch, mλdein, nmixx, odd youth, oh my girl, olivia marsh, primrose, purple kiss, red velvet, rescene, rolling quartz, rosé, say my name, seulgi, solar, soorin, spia, stayc, summer cake, taeyeon, triples, twice, uau, udtt, unis, viviz, vvs, vvup, wendy, wjsn, wooah, xg, yeji, yena, you dayeon, young posse, yuju, yuqi, yves"

	fmt.Println("Тестирование парсинга артистов:")
	fmt.Printf("Входная строка: %s\n\n", testInput)

	artists := parseArtists(testInput)

	fmt.Printf("Найдено артистов: %d\n", len(artists))
	fmt.Println("Список артистов:")
	for i, artist := range artists {
		fmt.Printf("%3d. %s\n", i+1, artist)
	}

	// Тестируем различные варианты ввода
	testCases := []string{
		"ablume, aespa, apink",
		"ablume,aespa,apink",
		"ablume,  aespa  ,  apink  ",
		"single artist",
		"artist with spaces, another artist",
		"",
		",,,",
	}

	fmt.Println("\n\nТестирование различных вариантов ввода:")
	for i, testCase := range testCases {
		fmt.Printf("\nТест %d: '%s'\n", i+1, testCase)
		result := parseArtists(testCase)
		fmt.Printf("Результат: %v (количество: %d)\n", result, len(result))
	}
}

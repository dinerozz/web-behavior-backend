package ai_analytics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/dinerozz/web-behavior-backend/internal/entity"
	"net/http"
	strings "strings"
	"time"
)

type AIAnalyticsService struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

type OpenAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message Message `json:"message"`
}

func NewAIAnalyticsService(apiKey string) *AIAnalyticsService {
	return &AIAnalyticsService{
		apiKey:  apiKey,
		baseURL: "https://api.openai.com/v1/chat/completions",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *AIAnalyticsService) AnalyzeDomainUsage(ctx context.Context, domainsCount int, domains []string, deepWorkData entity.DeepWorkData, engagementRate float64, trackedHours float64) (*entity.DomainAnalysis, error) {
	prompt := s.buildPrompt(domainsCount, domains, deepWorkData, engagementRate, trackedHours)

	request := OpenAIRequest{
		Model: "gpt-4o",
		Messages: []Message{
			{
				Role:    "system",
				Content: s.getSystemPrompt(),
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.1,
		MaxTokens:   500,
	}

	response, err := s.callOpenAI(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI: %w", err)
	}

	// Очищаем ответ от markdown разметки
	cleanResponse := s.cleanJSONResponse(response)

	var analysis entity.DomainAnalysis
	if err := json.Unmarshal([]byte(cleanResponse), &analysis); err != nil {
		// Логируем ошибку для отладки
		fmt.Printf("Failed to parse AI response: %v\nRaw response: %s\n", err, response)

		// Возвращаем fallback с частичным анализом
		return &entity.DomainAnalysis{
			FocusLevel: s.DetermineFocusLevelFallback(domainsCount),
			//FocusInsight:    s.generateFocusInsightFallback(domainsCount, domains),
			WorkPattern:     "unknown",
			Recommendations: []string{},
			Analysis: entity.DetailedAnalysis{
				DomainBreakdown: entity.DomainBreakdown{
					WorkTools:     []string{},
					Development:   []string{},
					Research:      []string{},
					Communication: []string{},
					Distractions:  []string{},
				},
				ProductivityScore: entity.ProductivityScore{
					Overall:     0,
					Focus:       0,
					Efficiency:  0,
					Balance:     0,
					Explanation: "AI анализ недоступен",
				},
				BehaviorInsights: []string{"Анализ не выполнен из-за ошибки"},
				KeyFindings:      []string{"Базовые данные доступны без AI"},
			},
		}, nil
	}

	return &analysis, nil
}

func (s *AIAnalyticsService) cleanJSONResponse(response string) string {
	response = strings.ReplaceAll(response, "```json", "")
	response = strings.ReplaceAll(response, "```", "")
	response = strings.TrimSpace(response)

	if !strings.HasPrefix(response, "{") {
		if start := strings.Index(response, "{"); start != -1 {
			response = response[start:]
		}
	}

	response = s.fixIncompleteJSON(response)

	return response
}

func (s *AIAnalyticsService) fixIncompleteJSON(jsonStr string) string {
	openBraces := strings.Count(jsonStr, "{")
	closeBraces := strings.Count(jsonStr, "}")

	if openBraces > closeBraces {
		if strings.HasSuffix(strings.TrimSpace(jsonStr), `"explanation": "`) {
			jsonStr += `"Анализ прерван"`
		} else if strings.Contains(jsonStr, `"explanation": "`) && !strings.Contains(jsonStr, `"explanation": ""`) {
			lastQuote := strings.LastIndex(jsonStr, `"`)
			if lastQuote > 0 && !strings.HasSuffix(jsonStr[:lastQuote+1], `""`) {
				jsonStr += `"`
			}
		}

		for i := 0; i < openBraces-closeBraces; i++ {
			jsonStr += "}"
		}
	}

	return jsonStr
}

func (s *AIAnalyticsService) getSystemPrompt() string {
	return `Ты эксперт по анализу цифрового поведения и продуктивности. 

ЗАДАЧА: Дать детальный, но краткий анализ на основе конкретных данных.

КАТЕГОРИИ ДОМЕНОВ:
- work_tools: Jira, Slack, корпоративные системы, CRM
- development: localhost, GitHub, CodeSandbox, IDE, облачные платформы  
- research: Stack Overflow, документация, курсы, блоги разработчиков
- communication: Gmail, Telegram, LinkedIn, мессенджеры
- distractions: YouTube, соцсети, новости, развлекательный контент

ОЦЕНКИ (0-100):
- overall: общая продуктивность (engagement + deep work + focus)
- focus: на основе deep work rate и количества доменов
- efficiency: на основе engagement rate
- balance: баланс рабочих/отвлекающих доменов

ИНСАЙТЫ: Конкретные наблюдения с цифрами и пояснениями.

ФОРМАТ JSON (без markdown):
{
  "focus_level": "high|medium|low",
  "focus_insight": "Краткий вывод с цифрами",
  "work_pattern": "deep_focused|task_switching|research_heavy|communication_intensive|distracted",
  "recommendations": ["рекомендация с обоснованием"],
  "analysis": {
    "domain_breakdown": {
      "work_tools": ["список доменов"],
      "development": ["список доменов"],
      "research": ["список доменов"], 
      "communication": ["список доменов"],
      "distractions": ["список доменов"]
    },
    "productivity_score": {
      "overall": 85,
      "focus": 90,
      "efficiency": 80,
      "balance": 85,
      "explanation": "Высокие показатели благодаря X, но снижены из-за Y"
    },
    "behavior_insights": [
      "93% времени deep work на localhost - отличная концентрация",
      "22 домена за 4+ часа - высокая фрагментация внимания"
    ],
    "key_findings": [
      "Преобладает разработка (localhost + dev инструменты)",
      "Минимальные отвлечения на развлекательный контент"
    ]
  }
}`
}

func (s *AIAnalyticsService) buildPrompt(domainsCount int, domains []string, deepWorkData entity.DeepWorkData, engagementRate float64, trackedHours float64) string {
	return fmt.Sprintf(`ДАННЫЕ ДЛЯ АНАЛИЗА:

📊 ОСНОВНЫЕ МЕТРИКИ:
- Время работы: %.2f часов
- Engagement rate: %.1f%% (активность в минутах)
- Уникальных доменов: %d
- Deep work: %d сессий (%.1f часов, %.1f%% времени)
- Средняя deep work сессия: %.1f мин (макс: %.1f мин)

🌐 ПОСЕЩЕННЫЕ ДОМЕНЫ:
%s

🎯 DEEP WORK ДОМЕНЫ:
%s

ЗАДАЧА: Проанализируй паттерн работы, дай конкретные инсайты с цифрами и практичные рекомендации.`,
		trackedHours,
		engagementRate,
		domainsCount,
		deepWorkData.SessionsCount,
		deepWorkData.TotalHours,
		deepWorkData.DeepWorkRate,
		deepWorkData.AverageMinutes,
		deepWorkData.LongestMinutes,
		formatDomainsForPrompt(domains),
		formatTopDomainsForPrompt(deepWorkData.TopDomains))
}

func formatDomainsForPrompt(domains []string) string {
	if len(domains) == 0 {
		return "Нет данных"
	}

	var result strings.Builder
	for i, domain := range domains {
		if i > 0 {
			result.WriteString(", ")
		}
		result.WriteString(domain)

		// Группируем по 6 доменов в строку для читабельности
		if (i+1)%6 == 0 && i < len(domains)-1 {
			result.WriteString("\n")
		}
	}
	return result.String()
}

func formatTopDomainsForPrompt(topDomains []entity.DeepWorkDomain) string {
	if len(topDomains) == 0 {
		return "Нет deep work сессий"
	}

	var result strings.Builder
	for i, domain := range topDomains {
		if i > 0 {
			result.WriteString(", ")
		}
		result.WriteString(fmt.Sprintf("%s (%.1f мин)", domain.Domain, domain.Minutes))
	}
	return result.String()
}

func getTopDomain(topDomains []entity.DeepWorkDomain) string {
	if len(topDomains) > 0 {
		return topDomains[0].Domain
	}
	return "не определен"
}

func (s *AIAnalyticsService) callOpenAI(ctx context.Context, request OpenAIRequest) (string, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API returned status %d", resp.StatusCode)
	}

	var openAIResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return "", err
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return openAIResp.Choices[0].Message.Content, nil
}

// Fallback функция если AI недоступен
func (s *AIAnalyticsService) DetermineFocusLevelFallback(domainsCount int) string {
	switch {
	case domainsCount <= 5:
		return "high"
	case domainsCount <= 15:
		return "medium"
	default:
		return "low"
	}
}

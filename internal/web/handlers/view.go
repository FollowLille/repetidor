package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"repetidor/internal/domain"
	"repetidor/internal/storage"
)

func parsePage(name string) (*template.Template, error) {
	return template.New("layout").Funcs(template.FuncMap{"tr": translate, "language": domain.LanguageByCode}).ParseFiles(filepath.Join("web", "templates", "layout.html"), filepath.Join("web", "templates", name))
}

func locale(r *http.Request) string {
	if cookie, err := r.Cookie("repetidor_locale"); err == nil && cookie.Value == "ru" {
		return "ru"
	}
	return "en"
}

func activeCourseID(r *http.Request) int64 {
	cookie, err := r.Cookie("repetidor_track")
	if err != nil {
		cookie, err = r.Cookie("repetidor_course")
	}
	if err == nil {
		if id, err := strconv.ParseInt(cookie.Value, 10, 64); err == nil && id > 0 {
			return id
		}
	}
	return 1
}

func activeCourse(repo storage.CourseRepository, r *http.Request) domain.Course {
	course, err := repo.Get(r.Context(), activeCourseID(r))
	if err != nil {
		course, _ = repo.Get(r.Context(), 1)
	}
	return course
}

func pageData(r *http.Request, values map[string]any) map[string]any {
	values["Locale"] = locale(r)
	values["CurrentPath"] = r.URL.RequestURI()
	return values
}

func translate(lang, text string) string {
	if lang != "ru" {
		return text
	}
	if value, ok := russianUI[text]; ok {
		return value
	}
	return text
}

var russianUI = map[string]string{
	"Learn": "Обучение", "Arena": "Арена", "Vocabulary": "Словарь", "Progress": "Прогресс", "Import": "Импорт", "Settings": "Настройки",
	"Your daily Spanish practice": "Ежедневная языковая практика", "Turn vocabulary into": "Превращайте слова в", "instinct.": "интуицию.",
	"Start practice": "Начать тренировку", "Cards": "Карточки", "Direction": "Направление", "Answer style": "Формат ответа", "Topics": "Темы",
	"Practice arena": "Игровая арена", "Build your own": "Соберите свою", "Create a mixed game session": "Создайте смешанную игровую сессию",
	"Games": "Игры", "Number of cards": "Количество карточек", "Start custom session": "Начать свою сессию", "Pick a challenge": "Выберите испытание",
	"All vocabulary": "Весь словарь", "Rounds": "Раунды", "Play": "Играть", "Quick choice": "Быстрый выбор", "Missing letters": "Пропущенные буквы", "Unscramble": "Анаграмма", "Word match": "Найти пару",
	"Language courses": "Языковые курсы", "Interface language": "Язык интерфейса", "Create course": "Создать курс", "Target language": "Изучаемый язык", "Reference language": "Опорный язык", "Theory language": "Язык теории",
	"Import words": "Импорт слов", "Paste words": "Вставьте слова", "Upload CSV": "Загрузить CSV", "Destination topic": "Тема назначения", "Import now": "Импортировать", "Imported": "Добавлено", "Skipped duplicates": "Пропущено дублей",
	"Bring your vocabulary in seconds.": "Добавьте весь словарь за несколько секунд.", "Paste a list or upload a CSV file.": "Вставьте список или загрузите CSV-файл.", "Choose topic": "Выберите тему", "Invalid": "Ошибки",
	"Keep interface language separate from the language you study and the language used for explanations.": "Язык приложения не зависит от изучаемого языка и языка объяснений.", "Active course": "Активный курс", "New learning space": "Новое пространство обучения", "Name": "Название",
	"One pair per line. The third column becomes notes.": "Одна пара на строку. Третья колонка станет заметкой.", "CSV columns: source, translation, notes.": "Колонки CSV: слово, перевод, заметка.",
	"Upload Excel": "Загрузить Excel", "XLSX columns: source, translation, notes.": "Колонки XLSX: слово, перевод, заметка.",
	"Or create a new topic": "Или создайте новую тему",
	"Courses":               "Курсы", "Course": "Курс", "Chapter": "Глава", "Language track": "Языковое направление", "Language tracks": "Языковые направления", "Active language track": "Активное языковое направление",
	"Build a path, not a pile.": "Соберите маршрут, а не свалку.", "Arrange topics into ordered courses and smaller chapters.": "Объединяйте темы в упорядоченные курсы и небольшие главы.",
	"Your first course starts here.": "Ваш первый курс начинается здесь.", "Combine existing topics into a route and add chapters when it grows.": "Соберите маршрут из существующих тем и разбивайте его на главы.",
	"Course builder": "Конструктор курса", "Create a course or chapter": "Создать курс или главу", "Description": "Описание", "Position": "Позиция", "Parent course": "Родительский курс", "Topics in this course": "Темы курса", "Prerequisites": "Предварительные курсы", "Requires": "Нужно пройти", "Add to learning path": "Добавить в маршрут", "Create vocabulary topics first.": "Сначала создайте темы со словами.",
}

func safeNext(raw string) string {
	if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
		return raw
	}
	return "/"
}

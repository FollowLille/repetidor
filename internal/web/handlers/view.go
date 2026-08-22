package handlers

import (
	"html/template"
	"math/rand"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"repetidor/internal/domain"
	"repetidor/internal/storage"
)

func parsePage(name string) (*template.Template, error) {
	return template.New("layout").Funcs(template.FuncMap{"tr": translate, "language": domain.LanguageByCode, "hasID": hasID, "sameID": sameID, "shuffleTokens": shuffleTokens, "boardColors": boardColors, "textColors": textColors, "boardBackgrounds": boardBackgrounds, "boardColorLabel": boardColorLabel, "boardBackgroundLabel": boardBackgroundLabel}).ParseFiles(filepath.Join("web", "templates", "layout.html"), filepath.Join("web", "templates", name))
}

type sentenceToken struct {
	ID    int
	Value string
}

func shuffleTokens(values []string) []sentenceToken {
	result := make([]sentenceToken, len(values))
	for i, value := range values {
		result[i] = sentenceToken{ID: i, Value: value}
	}
	rand.Shuffle(len(result), func(i, j int) { result[i], result[j] = result[j], result[i] })
	same := len(result) > 1
	for i := range result {
		same = same && result[i].ID == i
	}
	if same {
		result = append(result[1:], result[0])
	}
	return result
}

func boardColors() []string      { return []string{"violet", "amber", "mint", "rose", "slate"} }
func textColors() []string       { return []string{"white", "amber", "mint", "rose", "violet"} }
func boardBackgrounds() []string { return []string{"dots", "grid", "paper", "midnight", "sand"} }
func boardColorLabel(locale, value string) string {
	labels := map[string][2]string{"white": {"White", "Белый"}, "amber": {"Amber", "Янтарный"}, "mint": {"Mint", "Мятный"}, "rose": {"Rose", "Розовый"}, "violet": {"Violet", "Фиолетовый"}, "slate": {"Slate", "Графитовый"}}
	label := labels[value]
	if locale == "ru" {
		return label[1]
	}
	return label[0]
}
func boardBackgroundLabel(locale, value string) string {
	labels := map[string][2]string{"dots": {"Dots", "Точки"}, "grid": {"Grid", "Сетка"}, "paper": {"Paper", "Бумага"}, "midnight": {"Midnight", "Полночь"}, "sand": {"Sand", "Песок"}}
	label := labels[value]
	if locale == "ru" {
		return label[1]
	}
	return label[0]
}

func hasID(ids []int64, id int64) bool {
	for _, value := range ids {
		if value == id {
			return true
		}
	}
	return false
}
func sameID(value *int64, id int64) bool { return value != nil && *value == id }

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
	values["AuthorMode"] = authorMode(r)
	return values
}

func authorMode(r *http.Request) bool {
	cookie, err := r.Cookie("repetidor_workspace_mode")
	return err == nil && cookie.Value == "author"
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

func requiredMessage(r *http.Request, label string) string {
	if locale(r) == "ru" {
		return "Заполните поле «" + label + "»."
	}
	return label + " is required."
}

func topicExistsMessage(r *http.Request, name string) string {
	if locale(r) == "ru" {
		return "Тема «" + name + "» уже существует."
	}
	return "Topic \"" + name + "\" already exists."
}

func wordExistsMessage(r *http.Request, source, target string) string {
	if locale(r) == "ru" {
		return "Пара «" + source + " → " + target + "» уже есть в этой теме."
	}
	return "Word \"" + source + " → " + target + "\" already exists in this topic."
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
	"Destination topic (legacy files only)": "Тема назначения (только для старых файлов)", "Neutral columns: topic, target, reference, notes. Legacy files use the destination topic.": "Нейтральные колонки: тема, изучаемое слово, перевод, заметка. Старые файлы используют выбранную тему.",
	"Upload Excel": "Загрузить Excel", "XLSX columns: source, translation, notes.": "Колонки XLSX: слово, перевод, заметка.",
	"Or create a new topic": "Или создайте новую тему",
	"Courses":               "Курсы", "Course": "Курс", "Chapter": "Глава", "Language track": "Языковое направление", "Language tracks": "Языковые направления", "Active language track": "Активное языковое направление",
	"Build a path, not a pile.": "Соберите маршрут, а не свалку.", "Arrange topics into ordered courses and smaller chapters.": "Объединяйте темы в упорядоченные курсы и небольшие главы.",
	"Your first course starts here.": "Ваш первый курс начинается здесь.", "Combine existing topics into a route and add chapters when it grows.": "Соберите маршрут из существующих тем и разбивайте его на главы.",
	"Course builder": "Конструктор курса", "Create a course or chapter": "Создать курс или главу", "Description": "Описание", "Position": "Позиция", "Parent course": "Родительский курс", "Topics in this course": "Темы курса", "Prerequisites": "Предварительные курсы", "Requires": "Нужно пройти", "Add to learning path": "Добавить в маршрут", "Create vocabulary topics first.": "Сначала создайте темы со словами.",
	"Your daily language practice": "Ежедневная языковая практика", "Build a focused session around the words, direction, and answer style you need today.": "Соберите тренировку из нужных слов, направления перевода и способа ответа.",
	"cards": "карточек", "Both directions": "Оба направления", "Type and build": "Ввод и сборка", "Type only": "Только ввод", "Build letters only": "Только сборка букв", "optional": "необязательно", "No topics yet": "Тем пока нет",
	"Quick start": "Быстрый старт", "Choose your focus": "Выберите цель", "Every mode uses the same learning history, tuned for a different kind of session.": "Все режимы используют общую историю обучения, но подходят для разных типов тренировок.",
	"Mixed": "Смешанный", "Due": "Пора повторить", "Hard": "Сложные", "Easy": "Лёгкие", "Random": "Случайный", "Build letters": "Собрать слово", "Type answer": "Ввести ответ",
	"Adaptive practice based on your progress": "Адаптивная практика на основе вашего прогресса", "Words ready for their next review": "Слова, которые пора повторить", "Focus on words with recent mistakes": "Слова с недавними ошибками", "Reinforce words you already know": "Закрепление уже знакомых слов", "Uniform shuffle across vocabulary": "Равномерная случайная выборка из словаря", "Assemble answers letter by letter": "Собирайте ответы по одной букве", "Practice free recall by typing": "Вспоминайте и вводите ответ самостоятельно",
	"In progress": "В процессе", "Continue training": "Продолжить тренировку", "Session": "Сессия", "of": "из", "Resume": "Продолжить", "Details": "Подробнее",
	"Same vocabulary. Four new challenges.": "Тот же словарь. Четыре новых испытания.", "Choose answers, restore missing letters, unscramble words, or find a match. Every result stays part of your learning history.": "Выбирайте ответы, восстанавливайте буквы, собирайте слова и находите пары. Каждый результат сохраняется в истории обучения.", "Enter arena": "Перейти на арену",
	"Learning signal": "Сигнал обучения", "See what is sticking": "Посмотрите, что запоминается", "Accuracy, streaks, difficult words, and every completed session.": "Точность, серии, сложные слова и все завершённые сессии.", "Explore progress": "Посмотреть прогресс", "Collections": "Коллекции", "Your topics": "Ваши темы", "Manage": "Управлять", "No topics yet.": "Тем пока нет.",
	"Translate into": "Переведите на", "Recall the word in": "Вспомните слово на",
	"Train the same words.": "Тренируйте те же слова.", "Change the challenge.": "Меняйте испытание.", "Short game sessions use your real vocabulary and feed every result back into your learning progress.": "Короткие игровые сессии используют ваш словарь, а каждый результат влияет на прогресс обучения.",
	"Four ways to play": "Четыре способа играть", "Fast, focused, useful.": "Быстро, сфокусированно, полезно.", "Every round is saved to your session history.": "Каждый раунд сохраняется в истории сессий.", "Choose several challenges and let them rotate through one vocabulary list.": "Выберите несколько испытаний — они будут чередоваться на одном списке слов.", "Playlist": "Плейлист",
	"Vocabulary topics": "Темы словаря", "Leave empty to use everything": "Оставьте пустым, чтобы использовать всё", "Add vocabulary topics first.": "Сначала добавьте темы словаря.", "From 1 to 50 cards": "От 1 до 50 карточек", "Game modes": "Игровые режимы", "Choose a topic and session size inside any game.": "В каждой игре можно выбрать тему и размер сессии.", "Topic": "Тема",
	"One learning history": "Одна история обучения", "Games are practice, not a separate score.": "Игры — это практика, а не отдельный счёт.", "Correct answers strengthen a word. Mistakes make it more likely to return in Mixed and Hard sessions. You can resume every unfinished game from the home page.": "Верные ответы закрепляют слово. После ошибок оно чаще возвращается в смешанных и сложных сессиях. Незавершённую игру можно продолжить с главной страницы.",
	"Pick the translation from four answers.": "Выберите перевод из четырёх вариантов.", "Restore a word from its outline and meaning.": "Восстановите слово по его форме и значению.", "Put shuffled letters back into the right word.": "Соберите правильное слово из перемешанных букв.", "Find the matching word among a wider field.": "Найдите подходящее слово среди нескольких вариантов.",
	"Open existing topic": "Открыть существующую тему", "Edit existing topic": "Изменить существующую тему", "Back to topics": "Назад к темам", "Back to home": "На главную", "Create topic": "Создать тему", "Existing topics": "Существующие темы",
	"Training statistics": "Статистика тренировок", "Words": "Слова", "Attempts": "Попытки", "Correct": "Верно", "Accuracy": "Точность", "Frequent mistakes": "Частые ошибки", "Words that most often need another attempt.": "Слова, которым чаще всего нужна ещё одна попытка.", "misses": "ошибок", "close typos": "почти верных", "skips": "пропусков",
	"Recent sessions": "Недавние сессии", "Status": "Статус", "active": "активна", "completed": "завершена", "abandoned": "прервана", "No training sessions yet.": "Тренировок пока нет.", "Seen": "Показов", "Streak": "Серия", "Pain": "Сложность", "No words yet. Add words to a topic to start tracking progress.": "Слов пока нет. Добавьте их в тему, чтобы начать отслеживать прогресс.",
	"mixed": "смешанный", "random": "случайный", "type": "ввод", "build": "сборка", "choice": "быстрый выбор", "cloze": "пропуски", "anagram": "анаграмма", "match": "пары", "arcade": "своя арена", "due": "пора повторить", "hard": "сложные", "easy": "лёгкие",
	"Train this topic": "Тренировать эту тему", "Train all words": "Тренировать все слова", "Edit topic": "Изменить тему", "Add word": "Добавить слово", "Notes": "Заметки", "Save word": "Сохранить слово", "Saved words": "Сохранённые слова", "Edit": "Изменить", "Delete this word?": "Удалить это слово?", "Remove from topic": "Убрать из темы", "No words yet. Add the first one above.": "Слов пока нет. Добавьте первое выше.",
	"Save changes": "Сохранить изменения", "Delete this topic?": "Удалить эту тему?", "Delete topic": "Удалить тему", "Back to topic": "Назад к теме", "Edit word": "Изменить слово",
	"Card": "Карточка", "Try again": "Попробуйте ещё раз", "Prompt": "Задание", "Your reply": "Ваш ответ", "Target": "Правильный ответ", "Edit distance": "Расстояние редактирования", "Retry this card later": "Повторить карточку позже", "Session complete": "Сессия завершена", "Wrong": "Ошибки", "Skipped": "Пропущено", "Repeat mistakes": "Повторить ошибки", "Start again": "Начать заново", "View statistics": "Посмотреть статистику", "Start mixed session": "Начать смешанную сессию", "Open topics": "Открыть темы", "Unscramble the translation using every letter.": "Соберите перевод, используя все буквы.", "Backspace": "Стереть букву", "Clear": "Очистить", "Check": "Проверить", "Skip": "Пропустить", "Don't know": "Не знаю", "Leave arena": "Покинуть арену",
	"That answer does not match yet.": "Ответ пока не совпадает.", "Very close — this looks like a typo.": "Очень близко — похоже на опечатку.", "Skipped — progress was not changed.": "Пропущено — прогресс не изменён.", "Marked as unknown — this word will receive more practice.": "Отмечено как незнакомое — слово будет появляться чаще.", "No difficult words right now.": "Сейчас нет сложных слов.",
	"All courses": "Все курсы", "Theory starts with one clear idea.": "Теория начинается с одной ясной идеи.", "Add a text, example, note, or table block.": "Добавьте текст, пример, заметку или таблицу.", "I have read the theory": "Я прочитал теорию", "Exercises": "Упражнения", "Delete": "Удалить", "Delete block": "Удалить блок", "exercises are ready": "упражнений готово", "Check the rule while it is still fresh.": "Проверьте правило, пока оно ещё свежо.", "Start theory practice": "Начать практику по теории",
	"Vocabulary practice": "Практика слов", "Practice words from this course": "Потренировать слова этого курса", "Start course practice": "Начать практику курса", "Author tools": "Инструменты автора", "Course settings": "Настройки курса", "Add theory block": "Добавить блок теории", "Block type": "Тип блока", "Title": "Заголовок", "Content": "Содержание", "Add block": "Добавить блок", "Add exercise": "Добавить упражнение", "Exercise type": "Тип упражнения", "Question": "Вопрос", "Answer options": "Варианты ответа", "Correct answer": "Правильный ответ", "Explanation after answer": "Объяснение после ответа", "Delete this course?": "Удалить этот курс?", "Delete course": "Удалить курс",
	"Boards and media": "Доски и медиа", "Map ideas visually": "Связывайте идеи визуально", "No boards yet.": "Досок пока нет.", "Board name": "Название доски", "Create board": "Создать доску", "Back to course": "Назад к курсу", "Learning board": "Учебная доска", "Center": "По центру", "Create": "Создать", "Media": "Медиа", "Connect cards": "Связать карточки", "Add card": "Добавить карточку", "Card type": "Тип карточки", "Text": "Текст", "Note": "Заметка", "Color": "Цвет", "Add to board": "Добавить на доску", "Short heading": "Короткий заголовок", "Write an idea, rule or example": "Напишите идею, правило или пример", "What is this?": "Что здесь изображено?", "Choose image or audio": "Выберите изображение или аудио", "Upload media": "Загрузить медиа", "Image or audio": "Изображение или аудио", "Upload": "Загрузить", "Board card": "Карточка доски", "Edit card": "Редактировать карточку", "Resize": "Изменить размер", "Close": "Закрыть", "Cancel": "Отмена", "Save": "Сохранить", "Drag empty space to pan. Drag cards to arrange ideas.": "Тяните пустое пространство для перемещения. Перетаскивайте карточки, чтобы выстроить идеи.", "Drop your first idea here.": "Добавьте сюда первую идею.",
	"Vocabulary builder": "Конструктор словаря", "Save a translation pair and meet it in every matching course.": "Сохраните пару переводов — она появится во всех подходящих курсах.", "Vocabulary structure": "Структура словаря", "Group words by a situation, idea or learning goal.": "Объединяйте слова по ситуации, смыслу или учебной цели.", "Turn separate topics into a clear learning route.": "Соберите отдельные темы в понятный учебный маршрут.", "Choose the language you learn and the language that helps you.": "Выберите изучаемый язык и язык, который помогает вам учиться.",
	"Board tools": "Инструменты доски", "Move": "Перемещение", "Draw": "Рисовать", "Arrow": "Стрелка", "Eraser": "Ластик", "Drawing color": "Цвет линии",
	"Move canvas and cards": "Двигать холст и карточки", "Pen": "Карандаш", "Highlighter": "Маркер", "Draw arrow": "Нарисовать стрелку", "Erase drawing": "Стереть линию", "Undo drawing": "Отменить рисунок", "Redo drawing": "Вернуть рисунок", "Connect two cards": "Связать две карточки", "Zoom out": "Уменьшить масштаб", "Zoom in": "Увеличить масштаб", "Hide creation panel": "Скрыть панель создания", "Block": "Блок", "Sticker": "Стикер", "Free text": "Свободный текст", "Block color": "Цвет блока", "Text color": "Цвет текста",
	"Show creation panel": "Показать панель создания", "Place free text": "Добавить свободный текст", "Canvas text": "Текст на холсте", "Add free text": "Добавить свободный текст", "Type and place it on the board": "Введите текст и разместите его на доске", "Place on board": "Разместить на доске",
	"Boards": "Доски", "All boards": "Все доски", "Board background": "Фон доски", "Visual workspace": "Визуальное пространство", "Boards beyond courses": "Доски вне курсов", "Collect ideas, grammar maps and study plans without attaching them to one course.": "Собирайте идеи, грамматические карты и учебные планы без привязки к одному курсу.", "Board": "Доска", "Open canvas": "Открыть холст", "No global boards yet.": "Общих досок пока нет.", "Create a canvas for ideas that belong everywhere.": "Создайте холст для идей, которые относятся сразу ко всему.", "New canvas": "Новый холст", "Create global board": "Создать общую доску",
	"Learning levels": "Уровни обучения", "Filter learning path": "Фильтр учебного маршрута", "All levels": "Все уровни",
	"Practice this section": "Практиковать этот раздел", "Course exercises": "Упражнения курса", "Mixed practice across the whole course.": "Смешанная практика по всему курсу.", "Theory section": "Раздел теории", "Whole course": "Весь курс", "Untitled block": "Блок без названия", "Accepted answers": "Допустимые ответы", "Separate with commas": "Разделяйте запятыми", "Accepted, check spelling": "Ответ принят, проверьте написание", "Build a sentence": "Собрать предложение", "Words or phrases, separated by commas": "Слова или фразы через запятую", "Clear sentence": "Очистить предложение",
	"correct": "верно", "Resume session": "Продолжить сессию", "Back to statistics": "Назад к статистике", "Abandon session": "Завершить сессию", "Card review": "Разбор карточек", "Pending": "Ожидает ответа", "Reply": "Ответ", "close typo": "почти верно", "edit": "правка", "skipped": "пропущено", "Session details": "Детали сессии",
	"Portable courses": "Переносимые курсы", "Move a complete course in one file.": "Перенесите целый курс одним файлом.", "Preview blocks, exercises and vocabulary before anything is saved.": "Проверьте блоки, упражнения и словарь до сохранения.", "Theory blocks": "Блоки теории", "Duplicates": "Дубли", "Preview ready": "Предпросмотр готов", "Import everything atomically": "Импортировать всё атомарно", "If any item fails, nothing will be saved.": "Если возникнет ошибка, ничего не сохранится.", "Import course": "Импорт курса", "Upload course file": "Загрузить файл курса", "Versioned Repetidor course JSON, up to 10 MB.": "Версионированный JSON курса Repetidor, до 10 МБ.", "or": "или", "Paste course JSON": "Вставить JSON курса", "Preview import": "Проверить импорт", "Export course": "Экспортировать курс",
	"Portable vocabulary": "Переносимый словарь", "Samples and exports": "Шаблоны и экспорт", "Use the same neutral columns in CSV and Excel.": "Используйте одинаковые нейтральные колонки в CSV и Excel.", "Download sample CSV": "Скачать пример CSV", "Export language track": "Экспортировать направление", "Export topic": "Экспортировать тему", "Choose course": "Выберите курс", "Export course vocabulary": "Экспортировать слова курса",
	"Workspace mode": "Режим работы", "Learner": "Ученик", "Author": "Автор",
	"Back to theory": "Назад к теории", "Theory practice": "Практика по теории", "correct attempts": "верных попыток", "Your answer": "Ваш ответ", "input": "ввод ответа", "gap": "заполнить пропуск", "sentence_builder": "сборка предложения",
	"text": "текст", "example": "пример", "note": "заметка", "table": "таблица", "Example": "Пример", "Table": "Таблица", "Multiple choice": "Выбрать вариант", "Fill the gap": "Заполнить пропуск",
	"Return token":    "Вернуть слово",
	"Course chapters": "Главы курса", "Continue the learning path": "Продолжить учебный маршрут",
}

func safeNext(raw string) string {
	if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
		return raw
	}
	return "/"
}

# Spanish First Steps

Первый большой контент-пак для Repetidor.

## Состав

- 8 portable `repetidor-course` JSON-файлов.
- 1 корневой маршрут `Spanish First Steps`.
- 7 дочерних курсов/глав.
- 22 словарные темы.
- 946 topic-word entries (863 уникальные пары target/reference).
- 22 блока теории.
- 732 упражнения: input, gap, choice и sentence_builder.
- Уровни A1–A2.

## Порядок импорта

Импортируйте файлы по имени: `00` → `07`.

Порядок важен, потому что дочерние курсы используют `parent_key`, а грамматические главы используют prerequisite keys.

## Содержание

1. Базовый словарь: люди и повседневность.
2. Базовый словарь: мир вокруг.
3. Основа предложения: местоимения, ser, estar/hay, вопросы, отрицание, артикли, согласование, числа, время.
4. Presente: правильные и неправильные глаголы, gustar, частые конструкции, `ir a + infinitivo`.
5. Возвратные глаголы, прямые/косвенные и комбинированные местоимения, предлоги, сравнения.
6. Pretérito Perfecto и неправильные причастия.
7. Pretérito Indefinido и Perfecto vs Indefinido.

## Stable keys

Все course/topic/theory/exercise keys стабильны. Vocabulary использует стабильные `es.vocab.*` keys; одинаковые слова могут быть связаны с несколькими темами без создания дублей.

## Seed через CLI

Из корня Repetidor:

```powershell
./content/spanish-first-steps/seed.ps1
```

Параметры:

```powershell
./content/spanish-first-steps/seed.ps1 -TrackId 1 -SqlitePath ./repetidor.sqlite3
./content/spanish-first-steps/seed.ps1 -Preview
```

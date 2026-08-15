# Repetidor

A personal Spanish learning app focused on vocabulary review, topics, and interactive practice.

## Overview

Repetidor is a local-first Spanish learning project built primarily for personal use.

The main goal is to make vocabulary review more flexible and useful than generic flashcard apps by combining:

- topic-based vocabulary
- short topic notes and summaries
- interactive review modes
- lightweight spaced repetition
- mistake-focused review flow

## Available now

- shared vocabulary organized into one or more topics
- configurable persistent training sessions
- Spanish to Russian, Russian to Spanish, and mixed directions
- typed answers and letter-building practice
- adaptive Mixed, Due, Hard, Easy, and uniform Random modes
- separate Skip and Don't know actions
- retry-later and repeat-mistakes flows
- Practice Arena with choice, missing-letter, anagram, and matching games
- per-word progress, frequent mistakes, accuracy, streaks, and session history
- responsive local web interface backed by SQLite

## Tech stack

- Go
- Chi
- HTML templates
- HTMX
- SQLite

## Status

The core vocabulary, training, arena, session, and statistics flows are working. Import from text, CSV, and Excel exists as groundwork but is not yet exposed in the web interface.

## Notes

This project is built primarily for learning and personal use.

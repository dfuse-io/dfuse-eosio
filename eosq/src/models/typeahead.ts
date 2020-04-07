export interface Suggestion {
  label: string
  key: string
  summary?: string
}

export interface SuggestionSection {
  id: string
  suggestions: Suggestion[]
}

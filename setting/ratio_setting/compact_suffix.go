package ratio_setting

import "strings"

const CompactModelSuffix = "-openai-compact"
const CompactWildcardModelKey = "*" + CompactModelSuffix

func WithCompactModelSuffix(modelName string) string {
	if strings.HasSuffix(modelName, CompactModelSuffix) {
		return modelName
	}
	return modelName + CompactModelSuffix
}

func WithoutCompactModelSuffix(modelName string) (string, bool) {
	if !strings.HasSuffix(modelName, CompactModelSuffix) {
		return modelName, false
	}
	return strings.TrimSuffix(modelName, CompactModelSuffix), true
}

func MatchingModelCandidates(name string) []string {
	candidates := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)
	add := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		candidates = append(candidates, candidate)
	}

	add(name)
	add(FormatMatchingModelName(name))

	if baseModel, ok := WithoutCompactModelSuffix(name); ok {
		add(baseModel)
		add(FormatMatchingModelName(baseModel))
	}

	return candidates
}

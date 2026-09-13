package mutationcanary

func decorate(value string, maximum int) string {
	if len(value) > maximum {
		return value[:maximum]
	}
	return "value:" + value
}

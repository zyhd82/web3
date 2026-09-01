package repository

const (
	defaultPageSize = 10  // page_size 缺省值
	maxPageSize     = 100 // page_size 上限，防止无限制查询
)

// normalizePage 按分页规范（page 从 1 开始，page_size 默认 10、上限 100）纠正非法输入。
func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	switch {
	case pageSize <= 0:
		pageSize = defaultPageSize
	case pageSize > maxPageSize:
		pageSize = maxPageSize
	}
	return page, pageSize
}

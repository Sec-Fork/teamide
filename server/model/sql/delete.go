package sqlModel

import (
	"fmt"
	"server/base"
)

type Delete struct {
	Database string   `json:"database,omitempty"` // 库名
	Table    string   `json:"table,omitempty"`    // 表名
	Wheres   []*Where `json:"wheres,omitempty"`   // 条件
}

func (this_ *Delete) GetSqlParam(data map[string]interface{}) (sqlParam base.SqlParam, err error) {
	wrapTable := WrapTableName(this_.Database, this_.Table)

	params := []interface{}{}

	sql := fmt.Sprint("DELETE FROM ", wrapTable)

	whereSql := ""
	var whereParams []interface{}
	whereSql, whereParams, err = getWhereSqlParam(data, this_.Wheres)
	if err != nil {
		return
	}
	if IsNotEmpty(whereSql) {
		sql += " WHERE " + whereSql
		params = append(params, whereParams...)
	}

	sqlParam.Sql = sql
	sqlParam.Params = params
	return
}

func (this_ *Delete) GetSqlParams(dataList ...map[string]interface{}) (sqlParams []base.SqlParam, err error) {
	if len(dataList) == 0 {
		return
	}
	for _, data := range dataList {
		var sqlParam base.SqlParam
		sqlParam, err = this_.GetSqlParam(data)
		if err != nil {
			return
		}
		sqlParams = append(sqlParams, sqlParam)
	}

	return
}

func (this_ *Delete) GetTableColumns() (tableColumns map[string][]string) {
	tableColumns = map[string][]string{}

	wrapTable := WrapTableName(this_.Database, this_.Table)

	var columns []string
	appendWhereColumns(this_.Wheres, &columns)

	tableColumns[wrapTable] = columns

	return
}

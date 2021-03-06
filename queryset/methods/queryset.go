package methods

import (
	"fmt"
	"go/token"
	"log"
	"strings"
	"unicode"

	"github.com/jirfag/go-queryset/parser"
	"github.com/jirfag/go-queryset/queryset/field"
)

const qsReceiverName = "qs"
const qsDbName = qsReceiverName + ".db"

type QsStructContext struct {
	s parser.ParsedStruct
}

func NewQsStructContext(s parser.ParsedStruct) QsStructContext {
	return QsStructContext{
		s: s,
	}
}

func (ctx QsStructContext) qsTypeName() string {
	return ctx.s.TypeName + "QuerySet"
}

func (ctx QsStructContext) FieldCtx(f field.Info) QsFieldContext {
	return QsFieldContext{
		f:               f,
		QsStructContext: ctx,
	}
}

// QsFieldContext is a query set field context
type QsFieldContext struct {
	f             field.Info
	operationName string

	QsStructContext
}

func (ctx QsFieldContext) fieldName() string {
	return ctx.f.Name
}

func (ctx QsFieldContext) fieldDBName() string {
	return ctx.f.DBName
}

func (ctx QsFieldContext) fieldTypeName() string {
	return ctx.f.TypeName
}

func (ctx QsFieldContext) onFieldMethod() onFieldMethod {
	return newOnFieldMethod(ctx.operationName, ctx.fieldName())
}

func (ctx QsFieldContext) chainedQuerySetMethod() chainedQuerySetMethod {
	return newChainedQuerySetMethod(ctx.qsTypeName())
}

// WithOperationName return ctx with changed operation's name
func (ctx QsFieldContext) WithOperationName(operationName string) QsFieldContext {
	ctx.operationName = operationName
	return ctx
}

// retQuerySetMethod

type retQuerySetMethod struct {
	qsTypeName string
}

// GetReturnValuesDeclaration gets return values declaration
func (m retQuerySetMethod) GetReturnValuesDeclaration() string {
	return m.qsTypeName
}

func newRetQuerySetMethod(qsTypeName string) retQuerySetMethod {
	return retQuerySetMethod{
		qsTypeName: qsTypeName,
	}
}

// baseQuerySetMethod

type baseQuerySetMethod struct {
	structMethod
}

func newBaseQuerySetMethod(qsTypeName string) baseQuerySetMethod {
	return baseQuerySetMethod{
		structMethod: newStructMethod(qsReceiverName, qsTypeName),
	}
}

// chainedQuerySetMethod
type chainedQuerySetMethod struct {
	baseQuerySetMethod
	retQuerySetMethod
}

func newChainedQuerySetMethod(qsTypeName string) chainedQuerySetMethod {
	return chainedQuerySetMethod{
		baseQuerySetMethod: newBaseQuerySetMethod(qsTypeName),
		retQuerySetMethod:  newRetQuerySetMethod(qsTypeName),
	}
}

// FieldOperationNoArgsMethod is for unary operations: preload, orderby, etc
type FieldOperationNoArgsMethod struct {
	qsCallGormMethod
	onFieldMethod
	noArgsMethod
	chainedQuerySetMethod
}

func newFieldOperationNoArgsMethod(ctx QsFieldContext, transformFieldName bool) FieldOperationNoArgsMethod {

	gormArgName := ctx.fieldName()
	if transformFieldName {
		gormArgName = ctx.fieldDBName()
	}

	r := FieldOperationNoArgsMethod{
		onFieldMethod:         ctx.onFieldMethod(),
		qsCallGormMethod:      newQsCallGormMethod(ctx.operationName, `"%s"`, gormArgName),
		chainedQuerySetMethod: ctx.chainedQuerySetMethod(),
	}
	r.setFieldNameFirst(false) // UserPreload -> PreloadUser
	return r
}

// LowercaseFirstRune lowercases first rune of string
func LowercaseFirstRune(s string) string {
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

// commonInitialisms is a set of common initialisms.
// Only add entries that are highly unlikely to be non-initialisms.
// For instance, "ID" is fine (Freudian code is rare), but "AND" is not.
// XXX: copy-pasted from golint.
var commonInitialisms = map[string]bool{
	"ACL":   true,
	"API":   true,
	"ASCII": true,
	"CPU":   true,
	"CSS":   true,
	"DNS":   true,
	"EOF":   true,
	"GUID":  true,
	"HTML":  true,
	"HTTP":  true,
	"HTTPS": true,
	"ID":    true,
	"IP":    true,
	"JSON":  true,
	"LHS":   true,
	"QPS":   true,
	"RAM":   true,
	"RHS":   true,
	"RPC":   true,
	"SLA":   true,
	"SMTP":  true,
	"SQL":   true,
	"SSH":   true,
	"TCP":   true,
	"TLS":   true,
	"TTL":   true,
	"UDP":   true,
	"UI":    true,
	"UID":   true,
	"UUID":  true,
	"URI":   true,
	"URL":   true,
	"UTF8":  true,
	"VM":    true,
	"XML":   true,
	"XMPP":  true,
	"XSRF":  true,
	"XSS":   true,
}

func fieldNameToArgName(fieldName string) string {
	if commonInitialisms[fieldName] {
		return fieldName
	}

	argName := LowercaseFirstRune(fieldName)
	if token.Lookup(argName).IsKeyword() {
		return argName + "Value"
	}
	return argName
}

// StructOperationOneArgMethod is for struct operations with one arg
type StructOperationOneArgMethod struct {
	namedMethod
	chainedQuerySetMethod
	oneArgMethod
	qsCallGormMethod
}

func newStructOperationOneArgMethod(name, argTypeName, qsTypeName string) StructOperationOneArgMethod {
	argName := strings.ToLower(name)
	return StructOperationOneArgMethod{
		namedMethod:           newNamedMethod(name),
		chainedQuerySetMethod: newChainedQuerySetMethod(qsTypeName),
		oneArgMethod:          newOneArgMethod(argName, argTypeName),
		qsCallGormMethod:      newQsCallGormMethod(name, argName),
	}
}

type qsCallGormMethod struct {
	callGormMethod
}

func (m qsCallGormMethod) GetBody() string {
	return wrapToGormScope(m.callGormMethod.GetBody())
}

func newQsCallGormMethod(name, argsFmt string, argsArgs ...interface{}) qsCallGormMethod {
	return qsCallGormMethod{
		callGormMethod: newCallGormMethod(name, fmt.Sprintf(argsFmt, argsArgs...), qsDbName),
	}
}

// BinaryFilterMethod is a binary filter method
type BinaryFilterMethod struct {
	chainedQuerySetMethod
	onFieldMethod
	oneArgMethod
	qsCallGormMethod
}

// NewBinaryFilterMethod create new binary filter method
func NewBinaryFilterMethod(ctx QsFieldContext) BinaryFilterMethod {
	argName := fieldNameToArgName(ctx.fieldName())
	return BinaryFilterMethod{
		onFieldMethod:         ctx.onFieldMethod(),
		oneArgMethod:          newOneArgMethod(argName, ctx.fieldTypeName()),
		chainedQuerySetMethod: ctx.chainedQuerySetMethod(),
		qsCallGormMethod: newQsCallGormMethod("Where", "\"`%s` %s\", %s",
			ctx.fieldDBName(), getWhereCondition(ctx.operationName), argName),
	}
}

// InFilterMethod filters with IN condition
type InFilterMethod struct {
	chainedQuerySetMethod
	onFieldMethod
	nArgsMethod
	qsCallGormMethod
}

// GetBody returns method's body
func (m InFilterMethod) GetBody() string {
	tmpl := `iArgs := []interface{}{%s}
	for _, arg := range %s {
		iArgs = append(iArgs, arg)
	}
	`
	return fmt.Sprintf(tmpl, m.getArgName(0), m.getArgName(1)) + m.qsCallGormMethod.GetBody()
}

func newInFilterMethodImpl(ctx QsFieldContext, operationName, sql string) InFilterMethod {
	ctx = ctx.WithOperationName(operationName)
	argName := fieldNameToArgName(ctx.fieldName())
	args := newNArgsMethod(
		newOneArgMethod(argName, ctx.fieldTypeName()),
		newOneArgMethod(argName+"Rest", "..."+ctx.fieldTypeName()),
	)
	return InFilterMethod{
		onFieldMethod:         ctx.onFieldMethod(),
		nArgsMethod:           args,
		chainedQuerySetMethod: ctx.chainedQuerySetMethod(),
		qsCallGormMethod: newQsCallGormMethod("Where", "\"`%s` %s (?)\", iArgs",
			ctx.fieldDBName(), sql),
	}
}

// NewInFilterMethod create new IN filter method
func NewInFilterMethod(ctx QsFieldContext) InFilterMethod {
	return newInFilterMethodImpl(ctx, "In", "IN")
}

// NewNotInFilterMethod create new NOT IN filter method
func NewNotInFilterMethod(ctx QsFieldContext) InFilterMethod {
	return newInFilterMethodImpl(ctx, "NotIn", "NOT IN")
}

func getWhereCondition(name string) string {
	nameToOp := map[string]string{
		"eq":  "=",
		"ne":  "!=",
		"lt":  "<",
		"lte": "<=",
		"gt":  ">",
		"gte": ">=",
	}
	op := nameToOp[name]
	if op == "" {
		log.Fatalf("no operation for filter %q", name)
	}

	return fmt.Sprintf("%s ?", op)
}

// UnaryFilterMethod represents unary filter
type UnaryFilterMethod struct {
	onFieldMethod
	noArgsMethod
	chainedQuerySetMethod
	qsCallGormMethod
}

func newUnaryFilterMethod(ctx QsFieldContext, op string) UnaryFilterMethod {
	r := UnaryFilterMethod{
		onFieldMethod: ctx.onFieldMethod(),
		qsCallGormMethod: newQsCallGormMethod("Where", `"%s %s"`,
			ctx.fieldDBName(), op),
		chainedQuerySetMethod: ctx.chainedQuerySetMethod(),
	}
	return r
}

// unaryFilerMethod

// SelectMethod is a select field (all, one, etc)
type SelectMethod struct {
	namedMethod
	oneArgMethod
	baseQuerySetMethod
	gormErroredMethod
}

func newSelectMethod(name, gormName, argTypeName, qsTypeName string) SelectMethod {
	return SelectMethod{
		namedMethod:        newNamedMethod(name),
		baseQuerySetMethod: newBaseQuerySetMethod(qsTypeName),
		oneArgMethod:       newOneArgMethod("ret", argTypeName),
		gormErroredMethod:  newGormErroredMethod(gormName, "ret", qsDbName),
	}
}

// GetUpdaterMethod creates GetUpdater method
type GetUpdaterMethod struct {
	baseQuerySetMethod
	namedMethod
	noArgsMethod
	constRetMethod
	constBodyMethod
}

// NewGetUpdaterMethod creates GetUpdaterMethod
func NewGetUpdaterMethod(qsTypeName, updaterTypeMethod string) GetUpdaterMethod {
	return GetUpdaterMethod{
		baseQuerySetMethod: newBaseQuerySetMethod(qsTypeName),
		namedMethod:        newNamedMethod("GetUpdater"),
		constRetMethod:     newConstRetMethod(updaterTypeMethod),
		constBodyMethod:    newConstBodyMethod("return New%s(%s)", updaterTypeMethod, qsDbName),
	}
}

// DeleteMethod creates Delete method
type DeleteMethod struct {
	baseQuerySetMethod
	namedMethod
	noArgsMethod
	gormErroredMethod
}

// NewDeleteMethod creates Delete method
func NewDeleteMethod(qsTypeName, structTypeName string) DeleteMethod {
	return DeleteMethod{
		baseQuerySetMethod: newBaseQuerySetMethod(qsTypeName),
		namedMethod:        newNamedMethod("Delete"),
		gormErroredMethod:  newGormErroredMethod("Delete", structTypeName+"{}", qsDbName),
	}
}

// CountMethod creates Count method
type CountMethod struct {
	baseQuerySetMethod
	namedMethod
	noArgsMethod
	constRetMethod
	constBodyMethod
}

// NewCountMethod returns new CountMethod
func NewCountMethod(qsTypeName string) CountMethod {
	return CountMethod{
		baseQuerySetMethod: newBaseQuerySetMethod(qsTypeName),
		namedMethod:        newNamedMethod("Count"),
		constRetMethod:     newConstRetMethod("(int, error)"),
		constBodyMethod: newConstBodyMethod(`var count int
			err := %s.Count(&count).Error
			return count, err`, qsDbName),
	}
}

// Concrete methods

// NewPreloadMethod creates new Preload method
func NewPreloadMethod(ctx QsFieldContext) FieldOperationNoArgsMethod {
	r := newFieldOperationNoArgsMethod(ctx.WithOperationName("Preload"), false)
	return r
}

// NewOrderAscByMethod creates new OrderBy method ascending
func NewOrderAscByMethod(ctx QsFieldContext) FieldOperationNoArgsMethod {
	r := newFieldOperationNoArgsMethod(ctx.WithOperationName("OrderAscBy"), true)
	r.setGormMethodName("Order")
	r.setGormMethodArgs(fmt.Sprintf(`"%s ASC"`, ctx.fieldDBName()))
	return r
}

// NewOrderDescByMethod creates new OrderBy method descending
func NewOrderDescByMethod(ctx QsFieldContext) FieldOperationNoArgsMethod {
	r := newFieldOperationNoArgsMethod(ctx.WithOperationName("OrderDescBy"), true)
	r.setGormMethodName("Order")
	r.setGormMethodArgs(fmt.Sprintf(`"%s DESC"`, ctx.fieldDBName()))
	return r
}

// NewLimitMethod creates Limit method
func NewLimitMethod(qsTypeName string) StructOperationOneArgMethod {
	return newStructOperationOneArgMethod("Limit", "int", qsTypeName)
}

// NewAllMethod creates All method
func NewAllMethod(structName, qsTypeName string) SelectMethod {
	return newSelectMethod("All", "Find", fmt.Sprintf("*[]%s", structName), qsTypeName)
}

// NewOneMethod creates One method
func NewOneMethod(structName, qsTypeName string) SelectMethod {
	r := newSelectMethod("One", "First", fmt.Sprintf("*%s", structName), qsTypeName)
	const doc = `// One is used to retrieve one result. It returns gorm.ErrRecordNotFound
	// if nothing was fetched`
	r.setDoc(doc)
	return r
}

// NewIsNullMethod create IsNull method
func NewIsNullMethod(ctx QsFieldContext) UnaryFilterMethod {
	return newUnaryFilterMethod(ctx.WithOperationName("IsNull"), "IS NULL")
}

// NewIsNotNullMethod create IsNotNull method
func NewIsNotNullMethod(ctx QsFieldContext) UnaryFilterMethod {
	return newUnaryFilterMethod(ctx.WithOperationName("IsNotNull"), "IS NOT NULL")
}

package codegen

import (
    "log"

    "llvm.org/llvm/bindings/go/llvm"
    "github.com/k3v/lyca/src/parser"
)

var PRIMITIVE_TYPES = map[string]llvm.Type {
    "int": llvm.Int32Type(), "char": llvm.Int8Type(), "float": llvm.FloatType(), "boolean": llvm.Int1Type(),
}

var null llvm.Value = llvm.Value{}

func (c *Codegen) getLLVMFuncType(ret parser.Node, params []*parser.VarDeclNode, obj llvm.Type) llvm.Type {
    p := make([]llvm.Type, 0)
    if obj != llvm.VoidType() {
        p = append(p, obj)
    }

    for _, v := range params {
        p = append(p, c.getLLVMType(v.Type))
    }

    return llvm.FunctionType(c.getLLVMType(ret), p, false)
}

func (c *Codegen) getLLVMType(node parser.Node) llvm.Type {
    switch t := node.(type) {
    /*
    case *FuncTypeNode:
    case *ArrayTypeNode:
    */
    case *parser.NamedTypeNode:
        name := t.Name.Value
        if prim, ok := PRIMITIVE_TYPES[name]; ok {
            return prim
        }

        if t, ok := c.templates[name]; ok {
            return llvm.PointerType(t.Type, 0)
        }
    case *parser.BinaryExprNode:
        return c.getLLVMType(t.Left)
    case *parser.CharLitNode:
        return PRIMITIVE_TYPES["char"]
    case *parser.BoolLitNode:
        return PRIMITIVE_TYPES["boolean"]
    case *parser.NumLitNode:
        if t.IsFloat {
            return PRIMITIVE_TYPES["float"]
        } else {
            return PRIMITIVE_TYPES["int"]
        }
    case *parser.StringLitNode:
        return llvm.PointerType(c.templates["string"].Type, 0)
    case *parser.VarAccessNode:
        if param := c.getCurrParam(t.Name.Value); !param.IsNil() {
            return param.Type()
        } else if t := c.scope.GetValue(t.Name.Value).Type(); t != llvm.VoidType() {
            return t
        }
    case *parser.ObjectAccessNode:
        obj   := c.getLLVMType(t.Object)
        tmpl  := c.templates[c.getStructFromPointer(obj)]

        return c.getLLVMType(tmpl.Values[tmpl.Variables[t.Member.Value]].Type)
    case *parser.CallExprNode:
        return c.getLLVMTypeOfCall(t)
    }

    return llvm.VoidType()
}

func (c *Codegen) getLLVMTypeOfCall(node *parser.CallExprNode) llvm.Type {
    switch t := node.Function.(type) {
        case *parser.VarAccessNode:
            return c.module.NamedFunction(t.Name.Value).Type().ReturnType()
        case *parser.ObjectAccessNode:
            tmpl := c.getStructFromPointer(c.getLLVMType(t.Object))

            return c.module.NamedFunction("-" + tmpl + "-" + t.Member.Value).Type().ElementType().ReturnType()
    }

    return llvm.VoidType()
}

func (c *Codegen) getStructFromPointer(t llvm.Type) string {
    return c.getUnderlyingType(t).StructName()
}

func (c *Codegen) getUnderlyingType(t llvm.Type) llvm.Type {
    for t.TypeKind() == llvm.PointerTypeKind {
        t = t.ElementType()
    }

    return t
}

func (c *Codegen) convert(val llvm.Value, t llvm.Type) llvm.Value {
    if val.Type() == t {
        return val
    }

    if val == c.scope.GetValue("null") {
        log.Println("Null conversion")
        return llvm.ConstPointerNull(t)
    }

    switch val.Type() {
    case PRIMITIVE_TYPES["int"]:
        if t == PRIMITIVE_TYPES["float"] {
            return c.builder.CreateSIToFP(val, t, "")
        }
    }

    return val
}

func (c *Codegen) unbox(val llvm.Value) llvm.Value {
    t := c.getUnderlyingType(val.Type())
    switch t {
    case c.templates["string"].Type:
        val = c.builder.CreateStructGEP(val, 0, "")
        val = c.builder.CreateLoad(val, "")
    }

    return val
}

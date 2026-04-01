## gtk-ai — compresión de tokens

gtk-ai está activo como PostToolUse hook. Intercepta la salida de Bash, grep, find, ls, git y herramientas MCP antes de que entre en el contexto, aplicando deduplicación y truncado cuando supera el tamaño útil.

La compresión es transparente: no necesitas adaptar tu comportamiento.

Para ver el ahorro acumulado en la sesión actual:

```bash
gtkai gain
```

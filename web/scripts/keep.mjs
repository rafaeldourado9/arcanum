// O Vite limpa a pasta de saida; o .gitkeep precisa existir para o `go:embed all:dist`
// compilar num clone novo, antes de alguem rodar o build do painel.
import { writeFileSync } from "node:fs";

writeFileSync(new URL("../../internal/dashboard/dist/.gitkeep", import.meta.url), "");

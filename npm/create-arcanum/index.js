#!/usr/bin/env node
"use strict";
// npm create arcanum [folder]  ==  npx arcanum-gateway init [folder]
const { init } = require("arcanum-gateway/lib/init");

try {
  init(process.argv[2] || "arcanum");
} catch (err) {
  console.error(`create-arcanum: ${err.message}`);
  process.exit(1);
}

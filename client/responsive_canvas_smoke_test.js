#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const gameFile = path.join(__dirname, 'game.js');
const source = fs.readFileSync(gameFile, 'utf8');

const methodMatch = source.match(/resizeCanvas\(\) \{([\s\S]*?)\n\s*\}\n\s*getOrderedPlayerIDs\(\) \{/);

if (!methodMatch) {
  console.error('Could not locate resizeCanvas method in client/game.js');
  process.exit(1);
}

const resizeCanvas = new Function(methodMatch[1]);

function runProfile(name, profile) {
  const result = {
    name,
    renderCalls: 0,
    canvas: {
      parentElement: { clientWidth: profile.clientWidth },
      style: {},
      width: 0,
      height: 0
    },
    ctx: {
      setTransform(...args) {
        this.transform = args;
      },
      imageSmoothingEnabled: false
    },
    logicalCanvasWidth: 800,
    logicalCanvasHeight: 600,
    render() {
      this.renderCalls += 1;
    }
  };

  global.window = {
    devicePixelRatio: profile.dpr,
    innerHeight: profile.innerHeight
  };

  resizeCanvas.call(result);

  const expectedScale = Math.min(
    profile.clientWidth / result.logicalCanvasWidth,
    Math.max(240, profile.innerHeight - 220) / result.logicalCanvasHeight
  );
  const expectsFallback = expectedScale < 1;

  const styleWidth = Number.parseInt(result.canvas.style.width, 10);
  const styleHeight = Number.parseInt(result.canvas.style.height, 10);
  const aspectRatio = styleWidth / styleHeight;
  const expectedRatio = 4 / 3;

  if (Math.abs(aspectRatio - expectedRatio) > 0.02) {
    throw new Error(`${name}: expected a 4:3 canvas, got ${result.canvas.style.width} x ${result.canvas.style.height}`);
  }

  if (styleWidth > profile.clientWidth) {
    throw new Error(`${name}: canvas width ${styleWidth}px exceeds container width ${profile.clientWidth}px`);
  }

  if (styleHeight > profile.innerHeight - 220) {
    throw new Error(`${name}: canvas height ${styleHeight}px exceeds viewport budget`);
  }

  if (result.canvas.width !== Math.floor(styleWidth * profile.dpr)) {
    throw new Error(`${name}: drawing buffer width did not scale with devicePixelRatio`);
  }

  if (result.canvas.height !== Math.floor(styleHeight * profile.dpr)) {
    throw new Error(`${name}: drawing buffer height did not scale with devicePixelRatio`);
  }

  if (result.renderCalls !== 1) {
    throw new Error(`${name}: expected one render pass, got ${result.renderCalls}`);
  }

  if (expectsFallback && !result.layoutWarning) {
    throw new Error(`${name}: expected a layout warning for the constrained viewport profile`);
  }

  if (!expectsFallback && result.layoutWarning) {
    throw new Error(`${name}: did not expect a layout warning for the roomy viewport profile`);
  }

  if (!result.ctx.imageSmoothingEnabled) {
    throw new Error(`${name}: image smoothing should be enabled after resize`);
  }

  return `${name}: ${styleWidth}x${styleHeight} @ ${profile.dpr}dpr`;
}

const profiles = [
  ['desktop', { clientWidth: 1200, innerHeight: 900, dpr: 1 }],
  ['tablet', { clientWidth: 740, innerHeight: 820, dpr: 1.5 }],
  ['phone-portrait', { clientWidth: 360, innerHeight: 720, dpr: 2 }]
];

const summary = profiles.map(([name, profile]) => runProfile(name, profile));
console.log('responsive canvas smoke test passed');
summary.forEach(line => console.log(line));
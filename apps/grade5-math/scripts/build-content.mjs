// Reproducible app projection; canonical taxonomy files are never modified.
import { readFileSync, writeFileSync } from 'node:fs';
import { createHash } from 'node:crypto';
const read = path => JSON.parse(readFileSync(new URL(path, import.meta.url)));
const sourceBytes = readFileSync(new URL('../../../data/topics.json', import.meta.url));
const source = JSON.parse(sourceBytes);
const dependencies = read('../../../data/dependencies.json').dependencies;
const topics = new Map(source.topics.map(t => [t.id, t]));
const indegree = new Map(source.topics.map(t => [t.id, 0]));
const next = new Map(source.topics.map(t => [t.id, []]));
for (const d of dependencies) { indegree.set(d.topicId, indegree.get(d.topicId) + 1); next.get(d.prerequisiteId).push(d.topicId); }
const queue = [...indegree].filter(([, n]) => !n).map(([id]) => id);
for (let i = 0; i < queue.length; i++) for (const id of next.get(queue[i])) { indegree.set(id, indegree.get(id) - 1); if (!indegree.get(id)) queue.push(id); }
if (queue.length !== topics.size) throw Error('Canonical graph has a cycle');
const modules = [
  ['decimal', '小数与运算', '01', '把小数的大小与变化看清楚', '0.1'],
  ['fraction', '分数与数感', '02', '认识同样多，比较谁更多', '½'],
  ['factor', '因数与倍数', '03', '发现整数之间的小联系', '×'],
  ['algebra', '式与方程', '04', '用字母表达，找到未知数', 'x'],
  ['geometry', '图形与测量', '05', '从平面到立体，量一量', '▱'],
  ['data', '统计与可能性', '06', '读懂数据，想想可能性', '▥'],
].map(([id, name, number, description, symbol]) => ({ id, name, number, description, symbol }));
const skillRows = [
  ['decimal-compare', 'decimal', '小数的大小比较', 'mt_933BohS9BH', '对齐数位，再从左往右比较。小数位数更多，不一定更大。'],
  ['decimal-scale', 'decimal', '小数乘除 10、100', 'mt_LlMl2PbaZe', '每个数字的数位发生变化；乘 10 是原来的 10 倍，除以 10 是原来的十分之一。'],
  ['fraction-equivalent', 'fraction', '等值分数', 'mt_ebPelt-qAl', '分子、分母同时乘或除以同一个非零数，分数的大小不变。'],
  ['fraction-compare', 'fraction', '分数的大小比较', 'mt_SXbZ3bC9z7', '先确认同一个整体，可以通分，也可以与二分之一比较。'],
  ['factor-identify', 'factor', '找一个数的因数', 'mt_FHIAv6dfhU', '能把一个整数整除的正整数，是它的因数。'],
  ['factor-common', 'factor', '找最小公倍数', 'mt_FHIAv6dfhU', '分别列出两个数的正倍数，找出最小的共同倍数。'],
  ['algebra-expression', 'algebra', '用字母表示数量关系', 'mt_i9rJbuFO3p', '字母可以表示未知或变化的数，先读清题目中数量之间的关系。'],
  ['algebra-equation', 'algebra', '求等式中的未知数', 'mt_3JgrHY221M', '等号两边表示相同的量；用相反的运算找到未知数，再代回检查。'],
  ['geometry-area', 'geometry', '三角形与平行四边形面积', 'mt_ML5t7n2-U8', '平行四边形面积是底乘对应的高；同底同高的三角形面积是它的一半。'],
  ['geometry-volume', 'geometry', '长方体的体积', 'mt_nTL-owFJTF', '想象一层放多少个单位小正方体，再看有几层：长 × 宽 × 高。'],
  ['data-chart', 'data', '读取条形图并比较', 'mt_r8XnXwRA6g', '先看对象、单位和刻度，再比较条形代表的数量。'],
  ['data-chance', 'data', '比较事件的可能性', 'mt_iFFKZd-Vgv', '每个球被摸到的机会相同时，数量多的颜色更可能被摸到，但不一定会摸到。'],
];
const skills = skillRows.map(([id, moduleId, name, topicId, tip]) => ({ id, moduleId, name, topicId, tip }));
const questions = [];
const purposes = ['screen', 'confirm', 'practice', 'practice', 'retest', 'retest'];
function add(skillId, index, prompt, answer, explanation, extra = {}) {
  questions.push({ id: `${skillId}-${index + 1}`, skillId, purpose: purposes[index], kind: 'number', prompt, answer: String(answer), explanation, ...extra });
}
const comparisons = [{ id: 'gt', text: '大于 ＞' }, { id: 'eq', text: '等于 ＝' }, { id: 'lt', text: '小于 ＜' }];
function compare(a, b) { const [an, ad = 1] = a.split('/').map(Number), [bn, bd = 1] = b.split('/').map(Number); return an * bd > bn * ad ? 'gt' : an * bd < bn * ad ? 'lt' : 'eq'; }
[['0.8', '0.75'], ['3.09', '3.9'], ['0.50', '0.5'], ['2.07', '2.7'], ['6.4', '6.35'], ['0.308', '0.38']].forEach(([a, b], i) => {
  const answer = compare(a, b);
  add('decimal-compare', i, `${a} 与 ${b} 比较，应该填哪个符号？`, answer, `把数位对齐、末尾补零后比较，${a} ${answer === 'gt' ? '＞' : answer === 'lt' ? '＜' : '＝'} ${b}。`, { kind: 'choice', options: comparisons });
});
[['3.45', '×', 10, '34.5'], ['8.6', '÷', 10, '0.86'], ['0.72', '×', 100, '72'], ['45.6', '÷', 100, '0.456'], ['0.306', '×', 100, '30.6'], ['27.4', '÷', 10, '2.74']].forEach(([a, op, b, answer], i) => add('decimal-scale', i, `${a} ${op} ${b} = ？`, answer, `${a} ${op} ${b} = ${answer}。${op === '×' ? '每个数字向更高的数位移动' : '每个数字向更低的数位移动'}${b === 10 ? '一' : '两'}位。`));
[[2, 3, 2], [3, 5, 3], [1, 4, 3], [4, 7, 2], [3, 8, 2], [5, 6, 3]].forEach(([a, b, k], i) => add('fraction-equivalent', i, `把 ${a}/${b} 的分母变成 ${b * k}，分子和分母要同时乘 ${k}。请写出得到的完整分数。`, `${a * k}/${b * k}`, `分子和分母同时乘 ${k}，得到 ${a * k}/${b * k}，与 ${a}/${b} 一样大。`, { kind: 'fraction', requiredDenominator: b * k }));
[['2/3', '3/4'], ['5/6', '3/4'], ['1/2', '3/6'], ['3/5', '2/3'], ['4/5', '7/10'], ['5/8', '2/3']].forEach(([a, b], i) => {
  const answer = compare(a, b);
  const [an, ad] = a.split('/').map(Number), [bn, bd] = b.split('/').map(Number);
  add('fraction-compare', i, `同一个整体下，${a} 与 ${b} 比较，应该填哪个符号？`, answer, `通分到分母 ${ad * bd}：${a} = ${an * bd}/${ad * bd}，${b} = ${bn * ad}/${ad * bd}，所以填“${comparisons.find(o => o.id === answer).text}”。`, { kind: 'choice', options: comparisons });
});
[[24, [5, 6, 7], 6], [35, [4, 5, 6], 5], [18, [3, 4, 5], 3], [32, [6, 7, 8], 8], [42, [5, 6, 8], 6], [45, [4, 6, 9], 9]].forEach(([n, choices, answer], i) => add('factor-identify', i, `下面哪个数是 ${n} 的因数？`, answer, `${n} ÷ ${answer} = ${n / answer}，没有余数，所以 ${answer} 是 ${n} 的因数。`, { kind: 'choice', options: choices.map(n => ({ id: String(n), text: String(n) })) }));
const gcd = (a, b) => b ? gcd(b, a % b) : a;
[[4, 6], [3, 8], [6, 9], [5, 10], [6, 8], [4, 10]].forEach(([a, b], i) => { const answer = a * b / gcd(a, b); add('factor-common', i, `${a} 和 ${b} 的最小公倍数是多少？`, answer, `${answer} 可以同时被 ${a} 和 ${b} 整除，是它们最小的正公倍数。`); });
[[3, 2], [4, 3], [5, 4], [2, 6], [6, 5], [7, 3]].forEach(([a, b], i) => add('algebra-expression', i, `一本练习本 a 元，小禾买了 ${a} 本，又买了一支 ${b} 元的铅笔。一共花多少元？`, 'correct', `${a} 本练习本一共 ${a}a 元，再加铅笔的 ${b} 元，合起来是 ${a}a + ${b} 元。`, { kind: 'choice', options: [{ id: 'sum', text: `a + ${a} + ${b}` }, { id: 'correct', text: `${a}a + ${b}` }, { id: 'all', text: `${a + b}a` }] }));
[[7, 19, false], [25, 9, true], [8, 23, false], [30, 11, true], [12, 31, false], [40, 17, true]].forEach(([a, b, reversed], i) => { const answer = reversed ? a - b : b - a; add('algebra-equation', i, reversed ? `${a} − x = ${b}，x = ？` : `x + ${a} = ${b}，x = ？`, answer, reversed ? `x = ${a} − ${b} = ${answer}。代回：${a} − ${answer} = ${b}。` : `x = ${b} − ${a} = ${answer}。代回：${answer} + ${a} = ${b}。`); });
[[8, 5, true], [6, 4, false], [10, 6, true], [9, 4, false], [12, 5, true], [7, 6, false]].forEach(([base, height, triangle], i) => add('geometry-area', i, `一个${triangle ? '三角形' : '平行四边形'}的底是 ${base} 厘米，对应的高是 ${height} 厘米。面积是多少平方厘米？`, base * height / (triangle ? 2 : 1), triangle ? `三角形面积 = 底 × 对应的高 ÷ 2 = ${base} × ${height} ÷ 2 = ${base * height / 2} 平方厘米。` : `平行四边形面积 = 底 × 对应的高 = ${base} × ${height} = ${base * height} 平方厘米。`, { unit: '平方厘米', figure: { type: triangle ? 'triangle' : 'parallelogram', base, height } }));
[[5, 3, 2], [4, 3, 3], [6, 2, 4], [8, 3, 2], [7, 2, 3], [5, 4, 3]].forEach(([a, b, c], i) => add('geometry-volume', i, `一个长方体长 ${a} 厘米、宽 ${b} 厘米、高 ${c} 厘米。体积是多少立方厘米？`, a * b * c, `一层有 ${a} × ${b} = ${a * b} 个单位小正方体，一共 ${c} 层。体积 = ${a} × ${b} × ${c} = ${a * b * c} 立方厘米。`, { unit: '立方厘米', figure: { type: 'cuboid', base: a, width: b, height: c } }));
[[12, 18, 15], [9, 15, 12], [16, 24, 20], [14, 20, 18], [18, 27, 24], [20, 32, 28]].forEach(([a, b, c], i) => add('data-chart', i, '图中是三个小组收集的废纸质量。第二组比第一组多收集多少千克？', b - a, `第二组 ${b} 千克，第一组 ${a} 千克，相差 ${b} − ${a} = ${b - a} 千克。`, { unit: '千克', chart: [{ label: '第一组', value: a }, { label: '第二组', value: b }, { label: '第三组', value: c }] }));
[[4, 2], [2, 5], [3, 3], [5, 1], [2, 6], [7, 3]].forEach(([r, b], i) => { const answer = r === b ? 'equal' : r > b ? 'red' : 'blue'; add('data-chance', i, `袋子里有 ${r} 个红球、${b} 个蓝球，球的大小和质地相同。摇匀后不看颜色摸一个，哪种说法正确？`, answer, r === b ? '红球和蓝球一样多，每个球被摸到的机会相同，所以摸到两种颜色的可能性相同。' : `${r > b ? '红' : '蓝'}球更多，所以更可能摸到${r > b ? '红' : '蓝'}球。“更可能”不代表“一定”。`, { kind: 'choice', options: [{ id: 'red', text: '摸到红球的可能性更大' }, { id: 'blue', text: '摸到蓝球的可能性更大' }, { id: 'equal', text: '摸到两种颜色的可能性相同' }] }); });
// Translated labels for the small route projection. Untranslated nodes keep
// the source English name rather than inventing a Chinese curriculum mapping.
const labels = {
  mt_IegHBHERVa: '十分位与百分位', mt_kdWoAel3Zl: '认识十分之一', mt_Ep7TDFuYUa: '数轴上的等值分数', mt_FP_mjXaq3B: '理解分数等值',
  mt_HnKbuCliNS: '比较简单分数', mt_K5jM7vlVhA: '乘法口诀', mt_nZkL5_XjRX: '因数对', mt_4WaKWECpcv: '列出算式', mt_jHgRQ4hR0g: '运算顺序',
  mt_ehGS_uVSJv: '加减互为逆运算', mt_oIzycTBeE4: '等号的意义', mt_d7XktBQPxm: '数单位小正方体', mt_XLP1IM3Qbb: '多位数乘法',
};
const translatedNames = {
  'Bar graphs': '条形统计图',
  'Factors, multiples, and primes (age 9+)': '用因数与倍数解决问题', 'Understanding fractions (age 9+)': '理解分数的关系',
  'Unknown in Addition & Subtraction': '加减等式中的未知数', 'Advanced Maths Vocabulary': '用准确的数学语言表达',
  'Volume as additive': '把体积分开再合起来', 'Comparing Decimals': '比较小数大小', 'Multiplying fractions (age 10+)': '分数乘法',
  'Reading and writing numbers (age 10+)': '读写更大的数', 'Factors, multiples, and primes': '因数、倍数与质数',
  'Patterns with Powers of Ten': '乘除十的数位规律', 'Simple Chance Experiments': '简单的随机实验', 'Using Simple Formulae': '使用简单公式',
  'Multiplying and dividing': '乘除 10、100、1000', 'Area of Triangles & Parallelograms': '三角形与平行四边形面积',
  'Place Value × 10 Pattern': '数位之间的十倍关系', 'Comparing fractions (age 9+)': '比较分数大小',
  'Picture & Bar Graphs': '认识图画统计图与条形图', 'Simplifying Fractions': '约分', 'Pictograms and tally charts': '图画统计与计数',
  'Equivalent fractions (age 9+)': '等值分数', 'Area and perimeter of rectangles': '长方形的面积与周长',
  'Connecting maths to real life': '生活里的数学', 'Area by splitting shapes': '把图形分开求面积',
  'Writing Algebraic Equations': '把数量关系列成等式', 'Likelihood Language': '描述事件的可能性',
  'Dividing by 10 and 100': '除以 10 和 100', 'Volume of cuboids': '长方体体积', 'Scaled picture and bar graphs': '带刻度的图画统计图与条形图',
  'Pictograms and tally charts (age 6+)': '读懂图画统计与计数',
};
const routeNodes = new Set();
const routes = skills.map(s => {
  if (!topics.has(s.topicId)) throw Error(`Unknown topic ${s.topicId}`);
  const prerequisites = dependencies.filter(d => d.topicId === s.topicId).sort((a, b) => (a.strength === 'hard' ? 0 : 1) - (b.strength === 'hard' ? 0 : 1)).slice(0, 3);
  const unlocks = dependencies.filter(d => d.prerequisiteId === s.topicId && topics.get(d.topicId).subject === 'Mathematics' && topics.get(d.topicId).ageRangeStart <= 11).slice(0, 1);
  for (const d of [...prerequisites, ...unlocks]) { routeNodes.add(d.topicId); routeNodes.add(d.prerequisiteId); }
  routeNodes.add(s.topicId);
  return { skillId: s.id, topicId: s.topicId, prerequisites, unlocks };
});
const nodes = [...routeNodes].sort().map(id => ({ id, name: labels[id.replaceAll('-', '_')] ?? translatedNames[topics.get(id).name] ?? topics.get(id).name, sourceName: topics.get(id).name }));
const projection = { taxonomyVersion: source.version, topicFileSha256: createHash('sha256').update(sourceBytes).digest('hex'), dependencyFileSha256: createHash('sha256').update(readFileSync(new URL('../../../data/dependencies.json', import.meta.url))).digest('hex'), nodes, routes };
const checking = process.argv.includes('--check');
const save = (name, value) => {
  const path = new URL(`../content/${name}.json`, import.meta.url), bytes = JSON.stringify(value, null, 2) + '\n';
  if (checking) { if (readFileSync(path, 'utf8') !== bytes) throw Error(`${name}.json is stale; run the content generator`); }
  else writeFileSync(path, bytes);
};
save('skills', { modules, skills }); save('questions', { version: '2026-10-22.1', questions }); save('taxonomy-map', projection);
console.log(`${checking ? 'Verified' : 'Generated'} ${modules.length} modules, ${skills.length} skills, ${questions.length} questions and ${nodes.length} graph nodes.`);

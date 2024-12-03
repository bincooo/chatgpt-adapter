/**
 * 该代码为 https://github.com/teralomaniac/clewd 中的片段
 */

'use strict';

var ROLE_PREFIXS = {
    user: 'Human',
    assistant: 'Assistant',
    example_user: 'H',
    example_assistant: 'A',
    system: 'SYSTEM'
  },
  HyperProcess = function HyperProcess(system, messages, claudeMode) {
    var hyperMerge = function hyperMerge(content, mergeDisable) {
        var splitContent = content.split(new RegExp("\\n\\n(".concat(ROLE_PREFIXS['assistant'], "|").concat(ROLE_PREFIXS['user'], "|").concat(ROLE_PREFIXS['system'], "):"), 'g'));
        content = splitContent[0] + splitContent.slice(1).reduce(function (acc, current, index, array) {
          var merge = index > 1 && current === array[index - 2] && (current === ROLE_PREFIXS['user'] && !mergeDisable.user || current === ROLE_PREFIXS['assistant'] && !mergeDisable.assistant || current === ROLE_PREFIXS['system'] && !mergeDisable.system);
          return acc + (index % 2 !== 0 ? current.trim() : "\n\n".concat(merge ? '' : "".concat(current, ": ")));
        }, '');
        return content;
      },
      hyperRegex = function hyperRegex(content, order) {
        var regexLog = '',
          matches = content.match(new RegExp("<regex(?: +order *= *".concat(order, ")").concat(order === 2 ? '?' : '', "> *\"(/?)(.*)\\1(.*?)\" *: *\"(.*?)\" *</regex>"), 'gm'));
        matches && matches.forEach(function (match) {
          try {
            var reg = /<regex(?: +order *= *\d)?> *"(\/?)(.*)\1(.*?)" *: *"(.*?)" *<\/regex>/.exec(match);
            regexLog += match + '\n';
            if (reg[3].includes('s')) {
              reg[2] = reg[2].replace(/([^\\])\./g, '$1[\\s\\S]');
              reg[3] = reg[3].replace('s', 'm');
            }
            content = content.replace(new RegExp(reg[2], reg[3]), JSON.parse("\"".concat(reg[4].replace(/\\?"/g, '\\"'), "\"")));
          } catch (_unused) {}
        });
        return [content, regexLog];
      },
      HyperPmtProcess = function HyperPmtProcess(content) {
        var regex1 = hyperRegex(content, 1);
        content = regex1[0], regexLogs += regex1[1];
        var mergeDisable = {
          all: content.includes('<|Merge Disable|>'),
          system: content.includes('<|Merge System Disable|>'),
          user: content.includes('<|Merge Human Disable|>'),
          assistant: content.includes('<|Merge Assistant Disable|>')
        };
        content = content.replace(new RegExp("(\\n\\n|^\\s*)(?<!\\n\\n(".concat(ROLE_PREFIXS['user'], "|").concat(ROLE_PREFIXS['assistant'], "):.*?)").concat(ROLE_PREFIXS['system'], ":\\s*"), 'gs'), '$1').replace(new RegExp("(\\n\\n|^\\s*)".concat(ROLE_PREFIXS['system'], ": *"), 'g'), mergeDisable.all || mergeDisable.user || mergeDisable.system ? '$1' : "\n\n".concat(ROLE_PREFIXS['user'], ": "));
        content = hyperMerge(content, mergeDisable);
        var splitContent = content.split(new RegExp("\\n\\n(?=".concat(ROLE_PREFIXS['assistant'], ":|").concat(ROLE_PREFIXS['user'], ":)"), 'g')),
          match;
        while ((match = /<@(\d+)>([^]*?)<\/@\1>/g.exec(content)) !== null) {
          var index = splitContent.length - parseInt(match[1]) - 1;
          index >= 0 && (splitContent[index] += '\n\n' + match[2]);
          content = content.replace(match[0], '');
        }
        content = splitContent.join('\n\n').replace(/<@(\d+)>[^]*?<\/@\1>/g, '');
        var regex2 = hyperRegex(content, 2);
        content = regex2[0], regexLogs += regex2[1];
        content = hyperMerge(content, mergeDisable);
        var regex3 = hyperRegex(content, 3);
        content = regex3[0], regexLogs += regex3[1];
        content = content.replace(/<regex( +order *= *\d)?>.*?<\/regex>/gm, '').replace(/\r\n|\r/gm, '\n').replace(/\s*<\|curtail\|>\s*/g, '\n').replace(/\s*<\|join\|>\s*/g, '').replace(/\s*<\|space\|>\s*/g, ' ').replace(/<\|(\\.*?)\|>/g, function (match, p1) {
          try {
            return JSON.parse("\"".concat(p1, "\""));
          } catch (_unused2) {
            return match;
          }
        });
        return content.replace(/\s*<\|.*?\|>\s*/g, '\n\n').trim().replace(/^.+:/, '\n\n$&').replace(/(?<=\n)\n(?=\n)/g, '');
      };
    var prompt = system || '',
      regexLogs = '';
    messages.forEach(function (message) {
      var prefix = '\n\n' + (ROLE_PREFIXS[message.name] || ROLE_PREFIXS[message.role] + (message.name ? ": ".concat(message.name) : '')) + ': ';
      prompt += "".concat(prefix).concat(message.content.trim());
    });
    prompt = HyperPmtProcess(prompt);
    if (!claudeMode) prompt += "\n\n".concat(ROLE_PREFIXS['assistant'], ":");
    return {
      prompt: prompt,
      log: "\n####### Regex:\n".concat(regexLogs)
    };
  },
  ClaudePmtToMsgs = function ClaudePmtToMsgs(prompt, oai) {
    var rounds = prompt.split('\n\n' + ROLE_PREFIXS['user'] + ': ');
    return {
      messages: (oai && rounds.length > 1 ? [{
        role: 'system',
        content: rounds[0]
      }] : []).concat(rounds.slice(rounds.length > 1 && 1).flatMap(function (round) {
        var turns = round.split("\n\n".concat(ROLE_PREFIXS['assistant'], ":"));
        return [{
          role: 'user',
          content: turns[0].trim()
        }].concat(turns.slice(1).flatMap(function (turn) {
          return [{
            role: 'assistant',
            content: turn.trim()
          }];
        }));
      }).reduce(function (acc, current) {
        if (acc.length > 0 && (acc[acc.length - 1].role === current.role || !acc[acc.length - 1].content)) {
          acc[acc.length - 1].content += "\n\n".concat(ROLE_PREFIXS[current.role], ": ") + current.content;
        } else acc.push(current);
        return acc;
      }, [])),
      system: !oai && rounds.length > 1 ? rounds[0] : undefined
    };
  },
  CtoYmsgsConvert = function CtoYmsgsConvert(system, messages) {
    var YouMessages = [];
    if (messages[messages.length - 1].role !== 'assistant') messages.push({
      role: 'assistant',
      content: ''
    });
    while (messages.length > 1) YouMessages.unshift({
      answer: messages.pop().content.trim(),
      question: messages.pop().content.trim()
    });
    if (system) {
      if (system !== null && system !== void 0 && system.includes("\n\n".concat(ROLE_PREFIXS['assistant'], ":"))) {
        var segments = system.split("\n\n".concat(ROLE_PREFIXS['assistant'], ":"));
        YouMessages.unshift({
          question: segments[0].trim(),
          answer: segments.slice(1).join("\n\n".concat(ROLE_PREFIXS['assistant'], ":")).trim()
        });
      } else YouMessages[0].question = "".concat(system, "\n\n").concat(ROLE_PREFIXS['user'], ": ").concat(YouMessages[0].question).trim();
    }
    return YouMessages;
  },
  PmtToYouMsgs = function PmtToYouMsgs(prompt) {
    var _ClaudePmtToMsgs = ClaudePmtToMsgs(prompt, false),
      system = _ClaudePmtToMsgs.system,
      messages = _ClaudePmtToMsgs.messages;
    return CtoYmsgsConvert(system, messages);
  },
  youMsgToPmt = function youMsgToPmt(message) {
    var withPrefix = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : true;
    return (message.question.trim() && withPrefix ? "\n\n".concat(ROLE_PREFIXS['user'], ": ") : '') + message.question.trim() + (message.answer.trim() ? "\n\n".concat(ROLE_PREFIXS['assistant'], ": ").concat(message.answer.trim()) : '');
  },
  youPmtProcess = function youPmtProcess(prompt, ext) {
    var wedge = {
      txt: "\x9F",
      docx: "\x7F"
    };
    return prompt.split(new RegExp("\\n\\n(?=".concat(ROLE_PREFIXS['assistant'], ":|").concat(ROLE_PREFIXS['user'], ":)"), 'g')).join("\n".concat(wedge[ext], "\n"));
  };
(function (slice, mode) {
  var messagesClone = JSON.parse(JSON.stringify(slice));
  var _HyperProcess = HyperProcess("", messagesClone, true),
    prompt = _HyperProcess.prompt,
    log = _HyperProcess.log;
  console.log(log);
  var youPrompt = prompt.split(/\s*\[-youFileTag-\]\s*/);
  var filePrompt = youPrompt.pop().trim();
  var youMessages = [],
    youQuery = "";
  if (youPrompt.length > 0) {
    youMessages = PmtToYouMsgs(youPrompt.join('\n\n'));
    youQuery = youMsgToPmt(youMessages.pop(), false);
  }
  var chat = JSON.stringify(youMessages.map(function (message) {
    return {
      question: youPmtProcess(message.question, mode),
      answer: youPmtProcess(message.answer, mode)
    };
  }));
  return [{
    role: "messages",
    content: youPmtProcess(filePrompt, mode)
  }, {
    role: "chat",
    content: chat
  }, {
    role: "query",
    content: youPmtProcess(youQuery, mode)
  }];
})(messages, mode);

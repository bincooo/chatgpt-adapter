"use strict";

function _toConsumableArray(r) { return _arrayWithoutHoles(r) || _iterableToArray(r) || _unsupportedIterableToArray(r) || _nonIterableSpread(); }
function _nonIterableSpread() { throw new TypeError("Invalid attempt to spread non-iterable instance.\nIn order to be iterable, non-array objects must have a [Symbol.iterator]() method."); }
function _unsupportedIterableToArray(r, a) { if (r) { if ("string" == typeof r) return _arrayLikeToArray(r, a); var t = {}.toString.call(r).slice(8, -1); return "Object" === t && r.constructor && (t = r.constructor.name), "Map" === t || "Set" === t ? Array.from(r) : "Arguments" === t || /^(?:Ui|I)nt(?:8|16|32)(?:Clamped)?Array$/.test(t) ? _arrayLikeToArray(r, a) : void 0; } }
function _iterableToArray(r) { if ("undefined" != typeof Symbol && null != r[Symbol.iterator] || null != r["@@iterator"]) return Array.from(r); }
function _arrayWithoutHoles(r) { if (Array.isArray(r)) return _arrayLikeToArray(r); }
function _arrayLikeToArray(r, a) { (null == a || a > r.length) && (a = r.length); for (var e = 0, n = Array(a); e < a; e++) n[e] = r[e]; return n; }
/**
 * 该代码为 https://github.com/teralomaniac/clewd 中的片段
 */
var Config = {
  "PromptExperimentFirst": "",
  "PromptExperimentNext": "",
  "PersonalityFormat": "{{char}}'s personality: {{personality}}",
  "ScenarioFormat": "Dialogue scenario: {{scenario}}",
  "Settings": {
    "PromptExperiments": true,
    "AllSamples": false,
    "NoSamples": false,
    "StripAssistant": false,
    "StripHuman": false,
    "PassParams": true,
    "ClearFlags": true,
    "PreserveChats": false,
    "FullColon": true,
    "xmlPlot": true,
    "SkipRestricted": false,
    "padtxt": "1000,1000,15000"
  }
};
var Replacements = {
  user: 'Human',
  assistant: 'Assistant',
  system: '',
  example_user: 'H',
  example_assistant: 'A'
};
var genericFixes = function genericFixes(text) {
  return text.replace(/(\r\n|\r|\\n)/gm, '\n');
};
var xmlPlot_merge = function xmlPlot_merge(content, mergeTag, nonsys) {
    if (/(\n\n|^\s*)xmlPlot:\s*/.test(content)) {
      content = (nonsys ? content : content.replace(/(\n\n|^\s*)(?<!\n\n(Human|Assistant):[\s\S]*?)xmlPlot:\s*/g, '$1')).replace(/(\n\n|^\s*)xmlPlot: */g, mergeTag.system && mergeTag.human && mergeTag.all ? '\n\nHuman: ' : '$1');
    }
    mergeTag.all && mergeTag.human && (content = content.replace(/(?:\n\n|^\s*)Human:([\s\S]*?(?:\n\nAssistant:|$))/g, function (match, p1) {
      return '\n\nHuman:' + p1.replace(/\n\nHuman:\s*/g, '\n\n');
    }));
    mergeTag.all && mergeTag.assistant && (content = content.replace(/\n\nAssistant:([\s\S]*?(?:\n\nHuman:|$))/g, function (match, p1) {
      return '\n\nAssistant:' + p1.replace(/\n\nAssistant:\s*/g, '\n\n');
    }));
    return content;
  },
  xmlPlot_regex = function xmlPlot_regex(content, order) {
    var matches = content.match(new RegExp("<regex(?: +order *= *".concat(order, ")").concat(order === 2 ? '?' : '', "> *\"(/?)(.*)\\1(.*?)\" *: *\"(.*?)\" *</regex>"), 'gm'));
    matches && matches.forEach(function (match) {
      try {
        var reg = /<regex(?: +order *= *\d)?> *"(\/?)(.*)\1(.*?)" *: *"(.*?)" *<\/regex>/.exec(match);
        var reg2 = reg[2],
          reg3 = reg[3];
        if (reg3.includes('s')) {
          reg2 = reg2.replace(/([^\\])\./g, '$1[\\s\\S]');
          reg3 = reg3.replace('s', 'm');
        }
        content = content.replace(new RegExp(reg2, reg3), JSON.parse("\"".concat(reg[4].replace(/\\?"/g, '\\"'), "\"")));
      } catch (err) {
        console.log("\x1B[33mRegex error: \x1B[0m" + match + '\n' + err);
      }
    });
    return content;
  },
  xmlPlot = function xmlPlot(content) {
    var nonsys = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : false;
    //一次正则
    content = xmlPlot_regex(content, 1);
    //一次role合并
    var mergeTag = {
      all: !content.includes('<|Merge Disable|>'),
      system: !content.includes('<|Merge System Disable|>'),
      human: !content.includes('<|Merge Human Disable|>'),
      assistant: !content.includes('<|Merge Assistant Disable|>')
    };
    content = xmlPlot_merge(content, mergeTag, nonsys);
    //自定义插入
    var splitContent = content.split(/\n\n(?=Assistant:|Human:)/g),
      match;
    while ((match = /<@(\d+)>([\s\S]*?)<\/@\1>/g.exec(content)) !== null) {
      var index = splitContent.length - parseInt(match[1]) - 1;
      index >= 0 && (splitContent[index] += '\n\n' + match[2]);
      content = content.replace(match[0], '');
    }
    content = splitContent.join('\n\n').replace(/<@(\d+)>[\s\S]*?<\/@\1>/g, '');
    //二次正则
    content = xmlPlot_regex(content, 2);
    //二次role合并
    content = xmlPlot_merge(content, mergeTag, nonsys);

    //三次正则
    content = xmlPlot_regex(content, 3);
    //消除空XML tags、两端空白符和多余的\n
    content = content.replace(/<regex( +order *= *\d)?>[\s\S]*?<\/regex>/g, '').replace(/\r\n|\r/gm, '\n').replace(/\s*<\|curtail\|>\s*/g, '\n').replace(/\s*<\|join\|>\s*/g, '').replace(/\s*<\|space\|>\s*/g, ' ').replace(/\s*\n\n(H(uman)?|A(ssistant)?): +/g, '\n\n$1: ').replace(/<\|(\\.*?)\|>/g, function (match, p1) {
      try {
        return JSON.parse("\"".concat(p1.replace(/\\?"/g, '\\"'), "\""));
      } catch (e) {
        return match;
      }
    });

    //确保格式正确
    content = content.replace(/(\n\nHuman:(?![\s\S]*?\n\nAssistant:)[\s\S]*?|(?<!\n\nAssistant:[\s\S]*?))$/, '$&\n\nAssistant:').replace(/\s*<\|noAssistant\|>\s*([\s\S]*?)(?:\n\nAssistant:\s*)?$/, '\n\n$1');
    content.includes('<|reverseHA|>') && (content = content.replace(/\s*<\|reverseHA\|>\s*/g, '\n\n').replace(/Assistant|Human/g, function (match) {
      return match === 'Human' ? 'Assistant' : 'Human';
    }).replace(/\n(A|H): /g, function (match, p1) {
      return p1 === 'A' ? '\nH: ' : '\nA: ';
    }));
    return content.replace(Config.Settings.padtxt ? /\s*<\|(?!padtxt).*?\|>\s*/g : /\s*<\|.*?\|>\s*/g, '\n\n').trim().replace(/^.+:/, '\n\n$&').replace(/(?<=\n)\n(?=\n)/g, '');
  };
(function (messages) {
  var apiKey = true,
    stop_sequences;
  try {
    var _exec, _exec2;
    /************************* */
    var curPrompt = {
      firstUser: messages.find(function (message) {
        return 'user' === message.role;
      }),
      firstSystem: messages.find(function (message) {
        return 'system' === message.role;
      }),
      firstAssistant: messages.find(function (message) {
        return 'assistant' === message.role;
      }),
      lastUser: messages.findLast(function (message) {
        return 'user' === message.role;
      }),
      lastSystem: messages.findLast(function (message) {
        return 'system' === message.role && '[Start a new chat]' !== message.content;
      }),
      lastAssistant: messages.findLast(function (message) {
        return 'assistant' === message.role;
      })
    };
    var type = 'api';
    var _ref = function (messages) {
        var rgxScenario = /^\[Circumstances and context of the dialogue: ([\s\S]+?)\.?\]$/i,
          rgxPerson = /^\[([\s\S]+?)'s personality: ([\s\S]+?)\]$/i,
          messagesClone = JSON.parse(JSON.stringify(messages)),
          realLogs = messagesClone.filter(function (message) {
            return ['user', 'assistant'].includes(message.role);
          }),
          sampleLogs = messagesClone.filter(function (message) {
            return message.name;
          }),
          mergedLogs = [].concat(_toConsumableArray(sampleLogs), _toConsumableArray(realLogs));
        mergedLogs.forEach(function (message, idx) {
          var next = mergedLogs[idx + 1];
          message.customname = function (message) {
            return ['assistant', 'user'].includes(message.role) && null != message.name && !(message.name in Replacements);
          }(message);
          if (next) {
            if ('name' in message && 'name' in next) {
              if (message.name === next.name) {
                message.content += '\n' + next.content;
                next.merged = true;
              }
            } else if ('system' !== next.role) {
              if (next.role === message.role) {
                message.content += '\n' + next.content;
                next.merged = true;
              }
            } else {
              message.content += '\n' + next.content;
              next.merged = true;
            }
          }
        });
        var lastAssistant = realLogs.findLast(function (message) {
          return !message.merged && 'assistant' === message.role;
        });
        lastAssistant && Config.Settings.StripAssistant && (lastAssistant.strip = true);
        var lastUser = realLogs.findLast(function (message) {
          return !message.merged && 'user' === message.role;
        });
        lastUser && Config.Settings.StripHuman && (lastUser.strip = true);
        var systemMessages = messagesClone.filter(function (message) {
          return 'system' === message.role && !('name' in message);
        });
        systemMessages.forEach(function (message, idx) {
          var _message$content$matc;
          var scenario = (_message$content$matc = message.content.match(rgxScenario)) === null || _message$content$matc === void 0 ? void 0 : _message$content$matc[1],
            personality = message.content.match(rgxPerson);
          if (scenario) {
            message.content = Config.ScenarioFormat.replace(/{{scenario}}/gim, scenario);
            message.scenario = true;
          }
          if (3 === (personality === null || personality === void 0 ? void 0 : personality.length)) {
            message.content = Config.PersonalityFormat.replace(/{{char}}/gim, personality[1]).replace(/{{personality}}/gim, personality[2]);
            message.personality = true;
          }
          message.main = 0 === idx;
          message.jailbreak = idx === systemMessages.length - 1;
          ' ' === message.content && (message.discard = true);
        });
        Config.Settings.AllSamples && !Config.Settings.NoSamples && realLogs.forEach(function (message) {
          if (![lastUser, lastAssistant].includes(message)) {
            if ('user' === message.role) {
              message.name = message.customname ? message.name : 'example_user';
              message.role = 'system';
            } else if ('assistant' === message.role) {
              message.name = message.customname ? message.name : 'example_assistant';
              message.role = 'system';
            } else if (!message.customname) {
              throw Error('Invalid role ' + message.name);
            }
          }
        });
        Config.Settings.NoSamples && !Config.Settings.AllSamples && sampleLogs.forEach(function (message) {
          if ('example_user' === message.name) {
            message.role = 'user';
          } else if ('example_assistant' === message.name) {
            message.role = 'assistant';
          } else if (!message.customname) {
            throw Error('Invalid role ' + message.name);
          }
          message.customname || delete message.name;
        });
        var systems = [];
        var prompt = messagesClone.map(function (message, idx) {
          if (message.merged || message.discard) {
            return '';
          }
          if (message.content.length < 1) {
            return message.content;
          }
          var spacing = '';
          /******************************** */
          if (Config.Settings.xmlPlot) {
            idx > 0 && (spacing = '\n\n');
            var prefix = message.customname ? message.role + ': ' + message.name.replaceAll('_', ' ') + ': ' : 'system' !== message.role || message.name ? Replacements[message.name || message.role] + ': ' : 'xmlPlot: ' + Replacements[message.role];
            return "".concat(spacing).concat(message.strip ? '' : prefix).concat(message.content);
          } else {
            /******************************** */
            idx > 0 && (spacing = systemMessages.includes(message) ? '\n' : '\n\n');
            var _prefix = message.customname ? message.name.replaceAll('_', ' ') + ': ' : 'system' !== message.role || message.name ? Replacements[message.name || message.role] + ': ' : '' + Replacements[message.role];
            return "".concat(spacing).concat(message.strip ? '' : _prefix).concat('system' === message.role ? message.content : message.content.trim());
          } //
        });
        return {
          prompt: prompt.join(''),
          systems: systems
        };
      }(messages, type),
      prompt = _ref.prompt;

    /******************************** */
    var legacy = false,
      messagesAPI = !legacy && !/<\|completeAPI\|>/.test(prompt) || /<\|messagesAPI\|>/.test(prompt),
      fusion = true,
      wedge = '\r';
    var stopSet = (_exec = /<\|stopSet *(\[.*?\]) *\|>/.exec(prompt)) === null || _exec === void 0 ? void 0 : _exec[1],
      stopRevoke = (_exec2 = /<\|stopRevoke *(\[.*?\]) *\|>/.exec(prompt)) === null || _exec2 === void 0 ? void 0 : _exec2[1];
    if (stop_sequences || stopSet || stopRevoke) stop_sequences = JSON.parse(stopSet || '[]').concat(stop_sequences).concat(['\n\nHuman:', '\n\nAssistant:']).filter(function (item) {
      return !JSON.parse(stopRevoke || '[]').includes(item) && item;
    });
    prompt = Config.Settings.xmlPlot ? xmlPlot(prompt, legacy) : apiKey ? "\n\nHuman: ".concat(genericFixes(prompt), "\n\nAssistant:") : genericFixes(prompt).trim();
    Config.Settings.FullColon && (prompt = !legacy ? prompt.replace(fusion ? /\n(?!\nAssistant:\s*$)(?=\n(Human|Assistant):)/g : apiKey ? /(?<!\n\nHuman:[\s\S]*)\n(?=\nAssistant:)|\n(?=\nHuman:)(?![\s\S]*\n\nAssistant:)/g : /\n(?=\n(Human|Assistant):)/g, '\n' + wedge) : prompt.replace(fusion ? /(?<=\n\nAssistant):(?!\s*$)|(?<=\n\nHuman):/g : apiKey ? /(?<!\n\nHuman:[\s\S]*)(?<=\n\nAssistant):|(?<=\n\nHuman):(?![\s\S]*\n\nAssistant:)/g : /(?<=\n\n(Human|Assistant)):/g, '﹕'));

    /******************************** */
    var system;
    if (messagesAPI) {
      var rounds = prompt.replace(/^(?![\s\S]*\n\nHuman:)/, '\n\nHuman:').split('\n\nHuman:');
      messages = rounds.slice(1).flatMap(function (round) {
        var turns = round.split('\n\nAssistant:');
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
        if (Config.Settings.FullColon && acc.length > 0 && (acc[acc.length - 1].role === current.role || !acc[acc.length - 1].content)) {
          acc[acc.length - 1].content += (current.role === 'user' ? 'Human' : 'Assistant').replace(/.*/, legacy ? '\n$&﹕ ' : '\n' + wedge + '\n$&: ') + current.content;
        } else acc.push(current);
        return acc;
      }, []).filter(function (message) {
        return message.content;
      }), system = rounds[0].trim();
    }
    if (system) {
      return [{
        role: "system",
        content: system
      }].concat(_toConsumableArray(messages));
    }
    return messages;
  } catch (err) {
    throw err;
  }
})(messages);

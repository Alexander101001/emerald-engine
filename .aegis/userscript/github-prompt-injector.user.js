// ==UserScript==
// @name         Emerald Prompt Center Injector
// @namespace    http://emerald.engine
// @version      1.0.0
// @description  Injects a Prompt Center button next to the Code button on GitHub repo pages
// @author       Emerald Automaton
// @match        https://github.com/Alexander101001/*
// @match        https://github.com/*/emerald-engine*
// @match        https://github.com/*/emerald-saas*
// @grant        GM_openInTab
// @grant        GM_notification
// @run-at       document-idle
// ==/UserScript==

(function() {
    'use strict';

    var REPO_OWNER = 'Alexander101001';
    var WORKFLOW_FILE = 'prompt_control_center.yml';
    var BASE_URL = 'https://github.com/' + REPO_OWNER + '/emerald-engine/actions/workflows/' + WORKFLOW_FILE;
    var BUTTON_LABEL = 'Prompt Center';
    var BUTTON_ID = 'emerald-prompt-center-btn';

    function injectStyles() {
        var style = document.createElement('style');
        style.textContent = [
            '#' + BUTTON_ID + ' {',
            '  display: inline-flex;',
            '  align-items: center;',
            '  gap: 6px;',
            '  padding: 6px 14px;',
            '  font-size: 14px;',
            '  font-weight: 600;',
            '  font-family: -apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Oxygen,Ubuntu,sans-serif;',
            '  color: #ffffff;',
            '  background: linear-gradient(135deg, #059669, #10b981);',
            '  border: none;',
            '  border-radius: 6px;',
            '  cursor: pointer;',
            '  transition: all 0.2s ease;',
            '  box-shadow: 0 1px 3px rgba(0,0,0,0.15);',
            '  margin-left: 8px;',
            '  text-decoration: none;',
            '  line-height: 1;',
            '}',
            '#' + BUTTON_ID + ':hover {',
            '  background: linear-gradient(135deg, #047857, #059669);',
            '  box-shadow: 0 2px 6px rgba(0,0,0,0.25);',
            '  transform: translateY(-1px);',
            '}',
            '#' + BUTTON_ID + ':active {',
            '  transform: translateY(0);',
            '}',
            '#' + BUTTON_ID + ' .emerald-icon {',
            '  width: 16px;',
            '  height: 16px;',
            '  display: inline-block;',
            '}',
            '#' + BUTTON_ID + ' .emerald-icon svg {',
            '  width: 100%;',
            '  height: 100%;',
            '  fill: currentColor;',
            '}',
            '@media (max-width: 768px) {',
            '  #' + BUTTON_ID + ' {',
            '    padding: 5px 10px;',
            '    font-size: 12px;',
            '  }',
            '}',
        ].join('\n');
        document.head.appendChild(style);
    }

    function waitForElement(selector, callback, maxAttempts) {
        if (maxAttempts === undefined) maxAttempts = 20;
        var attempts = 0;
        var interval = setInterval(function() {
            attempts++;
            var el = document.querySelector(selector);
            if (el) {
                clearInterval(interval);
                callback(el);
                return;
            }
            if (attempts >= maxAttempts) {
                clearInterval(interval);
            }
        }, 500);
    }

    function createSVGIcon() {
        var span = document.createElement('span');
        span.className = 'emerald-icon';
        span.innerHTML = [
            '<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">',
            '<path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/>',
            '</svg>',
        ].join('');
        return span;
    }

    function createButton() {
        var existing = document.getElementById(BUTTON_ID);
        if (existing) existing.remove();

        var container = document.querySelector('ul[role="list"]') ||
                        document.querySelector('.file-navigation') ||
                        document.querySelector('.d-flex.gap-2') ||
                        document.querySelector('.BtnGroup');

        if (!container) return;

        var btn = document.createElement('a');
        btn.id = BUTTON_ID;
        btn.href = BASE_URL;
        btn.target = '_blank';
        btn.rel = 'noopener noreferrer';
        btn.title = 'Open Emerald Prompt Control Center';
        btn.appendChild(createSVGIcon());
        btn.appendChild(document.createTextNode(BUTTON_LABEL));

        btn.addEventListener('click', function(e) {
            e.preventDefault();
            if (typeof GM_openInTab === 'function') {
                GM_openInTab(BASE_URL, {active: true, insert: true});
            } else {
                window.open(BASE_URL, '_blank');
            }
            if (typeof GM_notification === 'function') {
                GM_notification({
                    text: 'Prompt Control Center opened in new tab.',
                    title: 'Emerald Engine',
                    timeout: 3000,
                });
            }
        });

        container.parentNode.insertBefore(btn, container.nextSibling);
    }

    function init() {
        var path = window.location.pathname;
        if (path.indexOf('Alexander101001') === -1 &&
            path.indexOf('emerald-engine') === -1 &&
            path.indexOf('emerald-saas') === -1) {
            return;
        }
        injectStyles();
        waitForElement(
            'ul[role="list"], .file-navigation, .d-flex.gap-2, .BtnGroup',
            function() {
                createButton();
            }
        );
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
    document.addEventListener('pjax:end', function() {
        setTimeout(init, 500);
    });
})();

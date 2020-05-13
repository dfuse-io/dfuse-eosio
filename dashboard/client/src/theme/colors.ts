/**
 * Copyright 2019 dfuse Platform Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/**
 * Name and general concept taken from Material UI Color System.
 *
 * @see https://material.io/design/color/the-color-system.html
 */

interface AppColor {
  [appName: string]: string;
}

const appColors: AppColor = {
  'abicodec': '#ffb230',
  'search-archiver': '#ff7165',
  'blockmeta': '#cc2644',
  'dashboard': '#600000',
  'dgraphql': '#8177e0',
  'search-indexer': '#c783ec',
  'trxdb-loader': '#219ce4',
  'search-live': '#76ddf0',
  'manager': '#00c0a2',
  'merger': '#a2e349',
  'mindreader': '#01f349',
  'relayer': '#657a90',
  'search-router': '#333333',
  'default': '#000000',
};


export const colors = {
  primary1: '#fff0f3',
  primary2: '#ffc6ce',
  primary3: '#ff9baa',
  primary4: '#ff7185',
  primary5: '#ff4660',
  primary6: '#e63652',
  primary7: '#cc2644',

  white: '#fff',
  transparancy: 'rgba(255,255,255,0)',

  ternary50: '#fafbfc',
  ternary100: '#f6f8f9',
  ternary200: '#f0f3f5',
  ternary250: '#e4eaef',
  ternary300: '#d8e1e9',
  ternary400: '#bbc7d3',
  ternary500: '#9fadbc',
  ternary600: '#8294a6',
  ternary700: '#657a90',
  ternary800: '#49617a',
  ternary900: '#2c4863',
  ternary950: '#203d5a',
  ternary1000: '#0f2e4d',
  ternary1100: '#0c243b',
  ternary1200: '#081929',

  link400: '#707bdb',
  link500: '#6673E5',
  link700: '#5a5ab4',

  highlight1: '#61d8c8',
  highlight2: '#34cfbd',

  alert1000: '#fbab0b',

  appColors: appColors,

  grey1: '#f8f8fa',
  grey2: '#dcdde2',
  grey3: '#c9cacf',
  grey4: '#b2b2b9',
  grey5: '#9a9ba3',
  grey6: '#7a7c84',
  grey7: '#5a5d65',
  grey8: '#3a3e47',
  grey9: '#1a1f28',
  grey10: '#000'
};

export type ColorTheme = typeof colors;
export const getAppColor = function(appName: any): string {
  return (colors.appColors[appName] ? colors.appColors[appName] : colors.appColors['default'])
};

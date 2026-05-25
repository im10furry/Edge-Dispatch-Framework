import type { ThemeConfig } from 'antd';
import { theme as antTheme } from 'antd';

const { darkAlgorithm } = antTheme;

const theme: ThemeConfig = {
  algorithm: darkAlgorithm,
  token: {
    colorPrimary: '#1f6feb',
    colorSuccess: '#3fb950',
    colorWarning: '#d29922',
    colorError: '#f85149',
    colorInfo: '#58a6ff',
    colorTextBase: '#e6edf3',
    colorBgBase: '#0a0e14',
    colorBgContainer: '#141b22',
    colorBgElevated: '#1c2733',
    colorBgLayout: '#0a0e14',
    colorBorder: '#30363d',
    colorBorderSecondary: '#21262d',
    borderRadius: 6,
    fontSize: 14,
    fontFamily: '"Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif',
  },
  components: {
    Layout: {
      bodyBg: '#0a0e14',
      headerBg: '#0d1117',
      siderBg: '#0d1117',
    },
    Menu: {
      darkItemBg: '#0d1117',
      darkSubMenuItemBg: '#0d1117',
      darkItemSelectedBg: '#1f6feb22',
      darkItemSelectedColor: '#58a6ff',
      itemBorderRadius: 6,
    },
    Table: {
      headerBg: '#141b22',
      rowHoverBg: '#1c2733',
      borderColor: '#30363d',
    },
    Card: {
      colorBgContainer: '#141b22',
    },
    Input: {
      colorBgContainer: '#0a0e14',
      colorBorder: '#30363d',
    },
    Select: {
      colorBgContainer: '#0a0e14',
      colorBorder: '#30363d',
    },
    Modal: {
      contentBg: '#1c2733',
      headerBg: '#1c2733',
    },
    Tooltip: {
      colorBgSpotlight: '#30363d',
    },
  },
};

export default theme;

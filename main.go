package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

const (
	screenWidth  = 450
	screenHeight = 600
	boardSize    = 4
	tileSize     = 100
	tileMargin   = 5
	boardMargin  = 20
)

// 游戏颜色
var (
	backgroundColor = color.RGBA{250, 248, 239, 255}
	boardColor      = color.RGBA{187, 173, 160, 255}
	emptyTileColor  = color.RGBA{205, 193, 180, 255}
	textColor       = color.RGBA{119, 110, 101, 255}
	textColorLight  = color.RGBA{249, 246, 242, 255}

	// 不同数值对应的颜色
	tileColors = map[int]color.RGBA{
		0:    {205, 193, 180, 255}, // 空白格
		2:    {238, 228, 218, 255},
		4:    {237, 224, 200, 255},
		8:    {242, 177, 121, 255},
		16:   {245, 149, 99, 255},
		32:   {246, 124, 95, 255},
		64:   {246, 94, 59, 255},
		128:  {237, 207, 114, 255},
		256:  {237, 204, 97, 255},
		512:  {237, 200, 80, 255},
		1024: {237, 197, 63, 255},
		2048: {237, 194, 46, 255},
		4096: {94, 218, 146, 255},
		8192: {57, 188, 120, 255},
	}
)

// 字体
var (
	normalFont font.Face
	boldFont   font.Face
	titleFont  font.Face
	scoreFont  font.Face
	chineseFont font.Face
)

// Game 代表游戏状态
type Game struct {
	board      [boardSize][boardSize]int
	score      int
	bestScore  int
	gameOver   bool
	win        bool
	showWin    bool
	message    string
	messageTime int
}

// 初始化游戏
func NewGame() *Game {
	g := &Game{
		score:     0,
		bestScore: 0,
		gameOver:  false,
		win:       false,
		showWin:   true,
	}
	g.initBoard()
	return g
}

// 初始化棋盘
func (g *Game) initBoard() {
	// 清空棋盘
	for i := 0; i < boardSize; i++ {
		for j := 0; j < boardSize; j++ {
			g.board[i][j] = 0
		}
	}

	// 添加两个初始方块
	g.addRandomTile()
	g.addRandomTile()
}

// 重置游戏
func (g *Game) resetGame() {
	g.score = 0
	g.gameOver = false
	g.win = false
	g.showWin = true
	g.initBoard()
}

// 添加随机方块
func (g *Game) addRandomTile() {
	// 找出所有空白格
	var emptyCells [][2]int
	for i := 0; i < boardSize; i++ {
		for j := 0; j < boardSize; j++ {
			if g.board[i][j] == 0 {
				emptyCells = append(emptyCells, [2]int{i, j})
			}
		}
	}

	// 如果没有空白格，返回
	if len(emptyCells) == 0 {
		return
	}

	// 随机选择一个空白格
	cell := emptyCells[rand.Intn(len(emptyCells))]
	i, j := cell[0], cell[1]

	// 90%的概率生成2，10%的概率生成4
	if rand.Float64() < 0.9 {
		g.board[i][j] = 2
	} else {
		g.board[i][j] = 4
	}
}

// 检查是否可以移动
func (g *Game) canMove() bool {
	// 检查是否有空白格
	for i := 0; i < boardSize; i++ {
		for j := 0; j < boardSize; j++ {
			if g.board[i][j] == 0 {
				return true
			}
		}
	}

	// 检查是否有相邻的相同数字
	for i := 0; i < boardSize; i++ {
		for j := 0; j < boardSize; j++ {
			// 检查右边
			if j < boardSize-1 && g.board[i][j] == g.board[i][j+1] {
				return true
			}
			// 检查下边
			if i < boardSize-1 && g.board[i][j] == g.board[i+1][j] {
				return true
			}
		}
	}

	return false
}

// 检查是否胜利
func (g *Game) checkWin() {
	for i := 0; i < boardSize; i++ {
		for j := 0; j < boardSize; j++ {
			if g.board[i][j] == 2048 {
				g.win = true
				return
			}
		}
	}
}

// 移动方向
const (
	DirectionUp = iota
	DirectionRight
	DirectionDown
	DirectionLeft
)

// 移动方块
func (g *Game) move(direction int) bool {
	moved := false
	
	// 根据方向进行移动
	switch direction {
	case DirectionUp:
		moved = g.moveUp()
	case DirectionRight:
		moved = g.moveRight()
	case DirectionDown:
		moved = g.moveDown()
	case DirectionLeft:
		moved = g.moveLeft()
	}

	// 如果有移动，添加一个随机方块
	if moved {
		g.addRandomTile()
		g.checkWin()
		
		// 检查游戏是否结束
		if !g.canMove() {
			g.gameOver = true
		}
	}

	return moved
}

// 向上移动
func (g *Game) moveUp() bool {
	moved := false

	for j := 0; j < boardSize; j++ {
		// 合并相同数字
		for i := 0; i < boardSize-1; i++ {
			for k := i + 1; k < boardSize; k++ {
				if g.board[k][j] == 0 {
					continue
				}
				if g.board[i][j] == 0 {
					g.board[i][j] = g.board[k][j]
					g.board[k][j] = 0
					i--
					moved = true
					break
				} else if g.board[i][j] == g.board[k][j] {
					g.board[i][j] *= 2
					g.score += g.board[i][j]
					if g.score > g.bestScore {
						g.bestScore = g.score
					}
					g.board[k][j] = 0
					moved = true
					break
				} else {
					break
				}
			}
		}

		// 移动所有方块
		for i := 0; i < boardSize-1; i++ {
			if g.board[i][j] == 0 {
				for k := i + 1; k < boardSize; k++ {
					if g.board[k][j] != 0 {
						g.board[i][j] = g.board[k][j]
						g.board[k][j] = 0
						moved = true
						break
					}
				}
			}
		}
	}

	return moved
}

// 向右移动
func (g *Game) moveRight() bool {
	moved := false

	for i := 0; i < boardSize; i++ {
		// 合并相同数字
		for j := boardSize - 1; j > 0; j-- {
			for k := j - 1; k >= 0; k-- {
				if g.board[i][k] == 0 {
					continue
				}
				if g.board[i][j] == 0 {
					g.board[i][j] = g.board[i][k]
					g.board[i][k] = 0
					j++
					moved = true
					break
				} else if g.board[i][j] == g.board[i][k] {
					g.board[i][j] *= 2
					g.score += g.board[i][j]
					if g.score > g.bestScore {
						g.bestScore = g.score
					}
					g.board[i][k] = 0
					moved = true
					break
				} else {
					break
				}
			}
		}

		// 移动所有方块
		for j := boardSize - 1; j > 0; j-- {
			if g.board[i][j] == 0 {
				for k := j - 1; k >= 0; k-- {
					if g.board[i][k] != 0 {
						g.board[i][j] = g.board[i][k]
						g.board[i][k] = 0
						moved = true
						break
					}
				}
			}
		}
	}

	return moved
}

// 向下移动
func (g *Game) moveDown() bool {
	moved := false

	for j := 0; j < boardSize; j++ {
		// 合并相同数字
		for i := boardSize - 1; i > 0; i-- {
			for k := i - 1; k >= 0; k-- {
				if g.board[k][j] == 0 {
					continue
				}
				if g.board[i][j] == 0 {
					g.board[i][j] = g.board[k][j]
					g.board[k][j] = 0
					i++
					moved = true
					break
				} else if g.board[i][j] == g.board[k][j] {
					g.board[i][j] *= 2
					g.score += g.board[i][j]
					if g.score > g.bestScore {
						g.bestScore = g.score
					}
					g.board[k][j] = 0
					moved = true
					break
				} else {
					break
				}
			}
		}

		// 移动所有方块
		for i := boardSize - 1; i > 0; i-- {
			if g.board[i][j] == 0 {
				for k := i - 1; k >= 0; k-- {
					if g.board[k][j] != 0 {
						g.board[i][j] = g.board[k][j]
						g.board[k][j] = 0
						moved = true
						break
					}
				}
			}
		}
	}

	return moved
}

// 向左移动
func (g *Game) moveLeft() bool {
	moved := false

	for i := 0; i < boardSize; i++ {
		// 合并相同数字
		for j := 0; j < boardSize-1; j++ {
			for k := j + 1; k < boardSize; k++ {
				if g.board[i][k] == 0 {
					continue
				}
				if g.board[i][j] == 0 {
					g.board[i][j] = g.board[i][k]
					g.board[i][k] = 0
					j--
					moved = true
					break
				} else if g.board[i][j] == g.board[i][k] {
					g.board[i][j] *= 2
					g.score += g.board[i][j]
					if g.score > g.bestScore {
						g.bestScore = g.score
					}
					g.board[i][k] = 0
					moved = true
					break
				} else {
					break
				}
			}
		}

		// 移动所有方块
		for j := 0; j < boardSize-1; j++ {
			if g.board[i][j] == 0 {
				for k := j + 1; k < boardSize; k++ {
					if g.board[i][k] != 0 {
						g.board[i][j] = g.board[i][k]
						g.board[i][k] = 0
						moved = true
						break
					}
				}
			}
		}
	}

	return moved
}

// 更新游戏状态
func (g *Game) Update() error {
	// 如果有消息，减少显示时间
	if g.messageTime > 0 {
		g.messageTime--
	} else {
		g.message = ""
	}

	// 处理按键输入
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		g.move(DirectionUp)
	} else if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		g.move(DirectionRight)
	} else if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		g.move(DirectionDown)
	} else if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		g.move(DirectionLeft)
	} else if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		// 重置游戏
		g.resetGame()
		g.showMessage("游戏已重置", 60)
	} else if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		// 如果已经赢了，继续游戏
		if g.win && g.showWin {
			g.showWin = false
			g.showMessage("继续游戏", 60)
		}
	}

	return nil
}

// 显示消息
func (g *Game) showMessage(msg string, time int) {
	g.message = msg
	g.messageTime = time
}

// 绘制游戏界面
func (g *Game) Draw(screen *ebiten.Image) {
	// 绘制背景
	screen.Fill(backgroundColor)

	// 绘制游戏标题
	text.Draw(screen, "2048", titleFont, screenWidth/2-50, 60, textColor)

	// 绘制分数
	drawScorePanel(screen, "分数", g.score, 30, 90)
	drawScorePanel(screen, "最高分", g.bestScore, screenWidth-100, 90)

	// 绘制游戏说明
	text.Draw(screen, "方向键移动 | R键重置", scoreFont, screenWidth/2-110, 150, textColor)

	// 绘制游戏棋盘
	drawBoard(screen, g.board)

	// 如果游戏胜利，显示胜利信息
	if g.win && g.showWin {
		drawOverlay(screen, "恭喜你赢了!", "按空格键继续游戏")
	}

	// 如果游戏结束，显示结束信息
	if g.gameOver {
		drawOverlay(screen, "游戏结束!", "按R键重新开始")
	}

	// 如果有消息，显示消息
	if g.message != "" {
		messageWidth := len(g.message) * 20
		ebitenutil.DrawRect(screen, float64(screenWidth/2-messageWidth/2-10), 180, float64(messageWidth+20), 40, color.RGBA{0, 0, 0, 180})
		text.Draw(screen, g.message, boldFont, screenWidth/2-messageWidth/2, 205, color.White)
	}
}

// 绘制分数面板
func drawScorePanel(screen *ebiten.Image, title string, score int, x, y int) {
	// 绘制背景
	ebitenutil.DrawRect(screen, float64(x), float64(y), 90, 60, boardColor)
	
	// 绘制标题
	text.Draw(screen, title, scoreFont, x+45-len(title)*4, y+20, textColorLight)
	
	// 绘制分数
	scoreText := fmt.Sprintf("%d", score)
	text.Draw(screen, scoreText, boldFont, x+45-len(scoreText)*7, y+45, textColorLight)
}

// 绘制棋盘
func drawBoard(screen *ebiten.Image, board [boardSize][boardSize]int) {
	// 棋盘位置
	boardX := (screenWidth - (tileSize*boardSize + tileMargin*(boardSize-1))) / 2
	boardY := 180

	// 绘制棋盘背景
	ebitenutil.DrawRect(screen, float64(boardX-boardMargin), float64(boardY-boardMargin), 
		float64((tileSize+tileMargin)*boardSize+boardMargin-tileMargin+boardMargin), 
		float64((tileSize+tileMargin)*boardSize+boardMargin-tileMargin+boardMargin), 
		boardColor)

	// 绘制每个格子
	for i := 0; i < boardSize; i++ {
		for j := 0; j < boardSize; j++ {
			x := boardX + j*(tileSize+tileMargin)
			y := boardY + i*(tileSize+tileMargin)
			
			// 获取格子颜色
			var tileColor color.RGBA
			if val, ok := tileColors[board[i][j]]; ok {
				tileColor = val
			} else {
				tileColor = tileColors[2048] // 超过2048的数字使用2048的颜色
			}
			
			// 绘制格子背景
			ebitenutil.DrawRect(screen, float64(x), float64(y), float64(tileSize), float64(tileSize), tileColor)
			
			// 如果不是空格，绘制数字
			if board[i][j] > 0 {
				numStr := fmt.Sprintf("%d", board[i][j])
				var tFace font.Face
				
				// 根据数字长度选择字体大小
				if board[i][j] < 100 {
					tFace = boldFont
				} else if board[i][j] < 1000 {
					tFace = boldFont
				} else {
					tFace = boldFont
				}
				
				// 修正: 更准确地计算文本位置使其居中
				bounds, _ := font.BoundString(tFace, numStr)
				textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
				textHeight := (bounds.Max.Y - bounds.Min.Y).Ceil()
				
				textX := x + (tileSize-textWidth)/2
				textY := y + (tileSize+textHeight)/2
				
				// 选择文本颜色
				textCol := textColor
				if board[i][j] > 4 {
					textCol = textColorLight
				}
				
				// 绘制数字
				text.Draw(screen, numStr, tFace, textX, textY, textCol)
			}
		}
	}
}

// 绘制覆盖层
func drawOverlay(screen *ebiten.Image, title, subtitle string) {
	// 绘制半透明背景
	ebitenutil.DrawRect(screen, 0, 0, float64(screenWidth), float64(screenHeight), color.RGBA{0, 0, 0, 180})
	
	// 绘制标题
	titleWidth := len(title) * 15
	text.Draw(screen, title, titleFont, screenWidth/2-titleWidth/2, screenHeight/2-40, color.White)
	
	// 绘制副标题
	subtitleWidth := len(subtitle) * 10
	text.Draw(screen, subtitle, boldFont, screenWidth/2-subtitleWidth/2, screenHeight/2+10, color.White)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	// 设置随机种子
	rand.Seed(time.Now().UnixNano())

	// 加载字体
	loadFonts()

	// 创建游戏
	game := NewGame()

	// 设置窗口标题
	ebiten.SetWindowTitle("2048游戏")
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowResizable(true)

	// 运行游戏
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

// 加载字体
func loadFonts() {
	// 加载默认字体
	tt, err := opentype.Parse(fonts.MPlus1pRegular_ttf)
	if err != nil {
		log.Fatal(err)
	}

	// 普通字体
	normalFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    12,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}

	// 粗体字体
	boldFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    16,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}

	// 标题字体
	titleFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    32,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}

	// 分数字体
	scoreFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    14,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}

	// 加载中文字体
	fontData, err := os.ReadFile("asset/zzgf_dianhei.otf")
	if err != nil {
		log.Printf("无法加载中文字体: %v", err)
		return
	}

	tt, err = opentype.Parse(fontData)
	if err != nil {
		log.Printf("无法解析中文字体: %v", err)
		return
	}

	chineseFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    16,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Printf("无法创建中文字体: %v", err)
		return
	}

	// 替换字体
	boldFont = chineseFont
	normalFont = chineseFont
	
	// 重新创建标题字体和分数字体
	titleFont, _ = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    32,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	
	scoreFont, _ = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    14,
		DPI:     72,
		Hinting: font.HintingFull,
	})
} 
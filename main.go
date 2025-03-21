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

// 移动动画的持续时间
const animationDuration = 10

// 方块动画状态
type TileAnimation struct {
	fromX, fromY int
	toX, toY     int
	value        int
	progress     float64
}

// Game 代表游戏状态
type Game struct {
	board             [boardSize][boardSize]int
	previousBoard     [boardSize][boardSize]int // 用于保存移动前的棋盘状态
	score             int
	bestScore         int
	gameOver          bool
	win               bool
	showWin           bool
	message           string
	messageTime       int
	animating         bool         // 是否正在执行动画
	animationProgress float64      // 动画进度 (0.0 - 1.0)
	animations        []TileAnimation // 方块动画列表
}

// 初始化游戏
func NewGame() *Game {
	g := &Game{
		score:            0,
		bestScore:        0,
		gameOver:         false,
		win:              false,
		showWin:          true,
		animating:        false,
		animationProgress: 0,
		animations:       []TileAnimation{},
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
	// 如果正在动画中，不处理输入
	if g.animating {
		return false
	}

	// 保存移动前的棋盘状态用于动画
	for i := 0; i < boardSize; i++ {
		for j := 0; j < boardSize; j++ {
			g.previousBoard[i][j] = g.board[i][j]
		}
	}
	
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

	// 如果有移动，添加一个随机方块并准备动画
	if moved {
		// 为移动的方块创建动画
		g.prepareAnimations()
		
		// 开始动画
		g.animating = true
		g.animationProgress = 0
		
		g.checkWin()
		
		// 检查游戏是否结束
		if !g.canMove() {
			g.gameOver = true
		}
	}

	return moved
}

// 准备方块移动动画
func (g *Game) prepareAnimations() {
	g.animations = []TileAnimation{}
	
	// 跟踪已经添加到动画列表的目标位置
	animatedTiles := make(map[[2]int]bool)
	
	// 遍历当前棋盘
	for i := 0; i < boardSize; i++ {
		for j := 0; j < boardSize; j++ {
			// 如果当前位置有方块
			if g.board[i][j] != 0 {
				// 查找这个方块在前一个状态的位置
				found := false
				
				// 如果是新生成的方块(在前一个状态中没有这个值)，跳过
				// 我们只为移动的方块创建动画
				
				// 查找可能的来源位置
				for pi := 0; pi < boardSize; pi++ {
					for pj := 0; pj < boardSize; pj++ {
						// 如果在前一个状态有相同值的方块(或者是合并的结果)
						if g.previousBoard[pi][pj] != 0 && 
							(g.previousBoard[pi][pj] == g.board[i][j] || 
							 g.previousBoard[pi][pj]*2 == g.board[i][j]) && 
							!(pi == i && pj == j) {
							
							// 确保我们还没有为这个目标位置创建动画
							posKey := [2]int{i, j}
							if !animatedTiles[posKey] {
								// 添加到动画列表
								g.animations = append(g.animations, TileAnimation{
									fromX:   pj,
									fromY:   pi,
									toX:     j,
									toY:     i,
									value:   g.previousBoard[pi][pj],
									progress: 0,
								})
								
								// 标记这个目标位置已经有动画了
								animatedTiles[posKey] = true
								found = true
								break
							}
						}
					}
					if found {
						break
					}
				}
			}
		}
	}
	
	// 添加随机生成的方块
	g.addRandomTile()
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

	// 更新动画状态
	if g.animating {
		g.animationProgress += 0.1
		if g.animationProgress >= 1.0 {
			g.animating = false
			g.animationProgress = 0
			g.animations = []TileAnimation{}
		}
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

	// 绘制游戏棋盘(只绘制背景和空格)
	drawBoard(screen, g.board)
	
	// 棋盘位置
	boardX := (screenWidth - (tileSize*boardSize + tileMargin*(boardSize-1))) / 2
	boardY := 180
	
	// 如果正在动画中，绘制动画方块
	if g.animating {
		// 绘制动画中的方块
		for _, anim := range g.animations {
			// 计算插值位置
			progress := easeOutQuad(g.animationProgress)
			currentX := float64(boardX) + 
				float64(anim.fromX)*(tileSize+tileMargin) + 
				float64(anim.toX-anim.fromX)*(tileSize+tileMargin)*progress
			currentY := float64(boardY) + 
				float64(anim.fromY)*(tileSize+tileMargin) + 
				float64(anim.toY-anim.fromY)*(tileSize+tileMargin)*progress
			
			// 获取方块颜色
			var tileColor color.RGBA
			if val, ok := tileColors[anim.value]; ok {
				tileColor = val
			} else {
				tileColor = tileColors[2048]
			}
			
			// 绘制方块
			ebitenutil.DrawRect(screen, currentX, currentY, float64(tileSize), float64(tileSize), tileColor)
			
			// 绘制数字
			numStr := fmt.Sprintf("%d", anim.value)
			var tFace font.Face
			
			// 根据数字长度选择字体大小
			if anim.value < 100 {
				tFace = boldFont
			} else if anim.value < 1000 {
				tFace = boldFont
			} else {
				tFace = boldFont
			}
			
			// 计算文本位置
			bounds, _ := font.BoundString(tFace, numStr)
			textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
			textHeight := (bounds.Max.Y - bounds.Min.Y).Ceil()
			
			textX := int(currentX) + (tileSize-textWidth)/2
			textY := int(currentY) + (tileSize+textHeight)/2
			
			// 选择文本颜色
			textCol := textColor
			if anim.value > 4 {
				textCol = textColorLight
			}
			
			// 绘制数字
			text.Draw(screen, numStr, tFace, textX, textY, textCol)
		}
		
		// 绘制静止的方块(非动画方块)
		for i := 0; i < boardSize; i++ {
			for j := 0; j < boardSize; j++ {
				// 跳过正在动画的方块位置
				isAnimating := false
				for _, anim := range g.animations {
					if anim.toX == j && anim.toY == i {
						isAnimating = true
						break
					}
				}
				
				if !isAnimating && g.board[i][j] > 0 {
					// 计算方块位置
					x := boardX + j*(tileSize+tileMargin)
					y := boardY + i*(tileSize+tileMargin)
					
					// 获取方块颜色
					var tileColor color.RGBA
					if val, ok := tileColors[g.board[i][j]]; ok {
						tileColor = val
					} else {
						tileColor = tileColors[2048]
					}
					
					// 绘制方块
					ebitenutil.DrawRect(screen, float64(x), float64(y), float64(tileSize), float64(tileSize), tileColor)
					
					// 绘制数字
					numStr := fmt.Sprintf("%d", g.board[i][j])
					var tFace font.Face
					
					if g.board[i][j] < 100 {
						tFace = boldFont
					} else if g.board[i][j] < 1000 {
						tFace = boldFont
					} else {
						tFace = boldFont
					}
					
					// 计算文本位置
					bounds, _ := font.BoundString(tFace, numStr)
					textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
					textHeight := (bounds.Max.Y - bounds.Min.Y).Ceil()
					
					textX := x + (tileSize-textWidth)/2
					textY := y + (tileSize+textHeight)/2
					
					// 选择文本颜色
					textCol := textColor
					if g.board[i][j] > 4 {
						textCol = textColorLight
					}
					
					// 绘制数字
					text.Draw(screen, numStr, tFace, textX, textY, textCol)
				}
			}
		}
	} else {
		// 正常绘制所有方块(非动画状态)
		for i := 0; i < boardSize; i++ {
			for j := 0; j < boardSize; j++ {
				if g.board[i][j] > 0 {
					// 计算方块位置
					x := boardX + j*(tileSize+tileMargin)
					y := boardY + i*(tileSize+tileMargin)
					
					// 获取方块颜色
					var tileColor color.RGBA
					if val, ok := tileColors[g.board[i][j]]; ok {
						tileColor = val
					} else {
						tileColor = tileColors[2048]
					}
					
					// 绘制方块
					ebitenutil.DrawRect(screen, float64(x), float64(y), float64(tileSize), float64(tileSize), tileColor)
					
					// 绘制数字
					numStr := fmt.Sprintf("%d", g.board[i][j])
					var tFace font.Face
					
					if g.board[i][j] < 100 {
						tFace = boldFont
					} else if g.board[i][j] < 1000 {
						tFace = boldFont
					} else {
						tFace = boldFont
					}
					
					// 计算文本位置
					bounds, _ := font.BoundString(tFace, numStr)
					textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
					textHeight := (bounds.Max.Y - bounds.Min.Y).Ceil()
					
					textX := x + (tileSize-textWidth)/2
					textY := y + (tileSize+textHeight)/2
					
					// 选择文本颜色
					textCol := textColor
					if g.board[i][j] > 4 {
						textCol = textColorLight
					}
					
					// 绘制数字
					text.Draw(screen, numStr, tFace, textX, textY, textCol)
				}
			}
		}
	}

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
			
			// 绘制空白格背景
			ebitenutil.DrawRect(screen, float64(x), float64(y), float64(tileSize), float64(tileSize), emptyTileColor)
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

// 缓动函数：缓出二次方
func easeOutQuad(t float64) float64 {
	return t * (2 - t)
} 
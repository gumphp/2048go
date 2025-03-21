package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io/ioutil"
	"log"
	"math"
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

// 方块动画类型
const (
	AnimationMove = iota // 移动动画
	AnimationMerge       // 合并动画
)

// 方块动画状态
type TileAnimation struct {
	fromX, fromY int
	toX, toY     int
	value        int
	progress     float64
	animType     int   // 动画类型
}

// 游戏进度文件路径
const saveFilePath = "2048_save.json"

// GameSave 用于保存游戏状态
type GameSave struct {
	Board     [boardSize][boardSize]int `json:"board"`
	Score     int                       `json:"score"`
	BestScore int                       `json:"best_score"`
	GameOver  bool                      `json:"game_over"`
	Win       bool                      `json:"win"`
	ShowWin   bool                      `json:"show_win"`
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
	lastMoveDirection int          // 最后一次移动的方向
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
		lastMoveDirection: -1,
	}
	
	// 尝试加载存档
	if !g.loadGame() {
		// 如果没有存档或加载失败，初始化新棋盘
		g.initBoard()
	}
	
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
	
	// 删除存档文件
	g.deleteSave()
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

	// 保存最后一次移动方向
	g.lastMoveDirection = direction

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
	
	// 记录已处理的目标格子，避免多个方块同时合并到同一个格子
	mergedCells := make(map[[2]int]bool)
	
	// 根据移动方向决定遍历顺序
	var rowOrder, colOrder []int
	
	// 初始化默认顺序
	rowOrder = make([]int, boardSize)
	colOrder = make([]int, boardSize)
	for i := 0; i < boardSize; i++ {
		rowOrder[i] = i
		colOrder[i] = i
	}
	
	// 根据移动方向调整遍历顺序
	// 这样可以确保先处理移动方向前面的方块，避免后面的方块"穿过"前面的方块
	switch g.lastMoveDirection {
	case DirectionUp:
		// 从上到下遍历
	case DirectionRight:
		// 从右到左遍历
		for i, j := 0, boardSize-1; i < j; i, j = i+1, j-1 {
			colOrder[i], colOrder[j] = colOrder[j], colOrder[i]
		}
	case DirectionDown:
		// 从下到上遍历
		for i, j := 0, boardSize-1; i < j; i, j = i+1, j-1 {
			rowOrder[i], rowOrder[j] = rowOrder[j], rowOrder[i]
		}
	case DirectionLeft:
		// 从左到右遍历
	}
	
	// 跟踪当前棋盘上有方块的格子
	currentPositions := make(map[[2]int]int) // 位置 -> 值
	for i := 0; i < boardSize; i++ {
		for j := 0; j < boardSize; j++ {
			if g.board[i][j] != 0 {
				currentPositions[[2]int{i, j}] = g.board[i][j]
			}
		}
	}
	
	// 遍历前一个状态的棋盘
	for _, ri := range rowOrder {
		for _, ci := range colOrder {
			i, j := ri, ci
			// 如果前一个状态该位置有方块
			if g.previousBoard[i][j] != 0 {
				// 如果当前位置没有方块或者值不同，说明方块发生了移动或合并
				if g.board[i][j] == 0 || g.board[i][j] != g.previousBoard[i][j] {
					// 查找方块移动的目标位置
					found := false
					prevValue := g.previousBoard[i][j]
					
					// 计算移动方向的搜索范围
					var rowRange, colRange []int
					switch g.lastMoveDirection {
					case DirectionUp:
						rowRange = make([]int, i+1)
						for r := 0; r <= i; r++ {
							rowRange[r] = r
						}
						colRange = []int{j}
					case DirectionRight:
						rowRange = []int{i}
						colRange = make([]int, boardSize-j)
						for c := 0; c < boardSize-j; c++ {
							colRange[c] = j + c
						}
					case DirectionDown:
						rowRange = make([]int, boardSize-i)
						for r := 0; r < boardSize-i; r++ {
							rowRange[r] = i + r
						}
						colRange = []int{j}
					case DirectionLeft:
						rowRange = []int{i}
						colRange = make([]int, j+1)
						for c := 0; c <= j; c++ {
							colRange[c] = c
						}
					}
					
					// 首先查找相同值的方块（移动的情况）
					for _, r := range rowRange {
						for _, c := range colRange {
							pos := [2]int{r, c}
							if val, exists := currentPositions[pos]; exists && val == prevValue {
								// 为这个方块创建移动动画
								g.animations = append(g.animations, TileAnimation{
									fromX:    j,
									fromY:    i,
									toX:      c,
									toY:      r,
									value:    prevValue,
									progress: 0,
									animType: AnimationMove,
								})
								
								// 从当前位置列表中删除，避免重复处理
								delete(currentPositions, pos)
								found = true
								break
							}
						}
						if found {
							break
						}
					}
					
					// 如果没找到相同值的方块，查找合并的情况
					if !found {
						for _, r := range rowRange {
							for _, c := range colRange {
								pos := [2]int{r, c}
								if val, exists := currentPositions[pos]; exists && val == prevValue*2 && !mergedCells[pos] {
									// 为这个方块创建合并动画
									g.animations = append(g.animations, TileAnimation{
										fromX:    j,
										fromY:    i,
										toX:      c,
										toY:      r,
										value:    prevValue,
										progress: 0,
										animType: AnimationMerge,
									})
									
									// 标记该位置已有合并动画
									mergedCells[pos] = true
									found = true
									break
								}
							}
							if found {
								break
							}
						}
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

// 保存游戏状态
func (g *Game) saveGame(showMessage bool) {
	// 创建保存对象
	save := GameSave{
		Board:     g.board,
		Score:     g.score,
		BestScore: g.bestScore,
		GameOver:  g.gameOver,
		Win:       g.win,
		ShowWin:   g.showWin,
	}

	// 将对象序列化为JSON
	data, err := json.Marshal(save)
	if err != nil {
		log.Printf("保存游戏失败: %v", err)
		if showMessage {
			g.showMessage("保存游戏失败", 60)
		}
		return
	}

	// 写入文件
	err = ioutil.WriteFile(saveFilePath, data, 0644)
	if err != nil {
		log.Printf("写入存档文件失败: %v", err)
		if showMessage {
			g.showMessage("保存游戏失败", 60)
		}
		return
	}

	if showMessage {
		g.showMessage("游戏已保存", 60)
	}
}

// 加载游戏状态
func (g *Game) loadGame() bool {
	// 检查文件是否存在
	if _, err := os.Stat(saveFilePath); os.IsNotExist(err) {
		g.showMessage("没有找到存档", 60)
		return false
	}

	// 读取文件
	data, err := ioutil.ReadFile(saveFilePath)
	if err != nil {
		log.Printf("读取存档文件失败: %v", err)
		g.showMessage("加载游戏失败", 60)
		return false
	}

	// 解析JSON
	var save GameSave
	err = json.Unmarshal(data, &save)
	if err != nil {
		log.Printf("解析存档数据失败: %v", err)
		g.showMessage("加载游戏失败", 60)
		return false
	}

	// 恢复游戏状态
	g.board = save.Board
	g.score = save.Score
	g.bestScore = save.BestScore
	g.gameOver = save.GameOver
	g.win = save.Win
	g.showWin = save.ShowWin

	g.showMessage("游戏已加载", 60)
	return true
}

// 删除存档
func (g *Game) deleteSave() {
	if _, err := os.Stat(saveFilePath); !os.IsNotExist(err) {
		err = os.Remove(saveFilePath)
		if err != nil {
			log.Printf("删除存档文件失败: %v", err)
			return
		}
	}
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
		g.animationProgress += 0.15  // 调快动画速度
		if g.animationProgress >= 1.0 {
			g.animating = false
			g.animationProgress = 0
			g.animations = []TileAnimation{}
		}
	}

	// 处理按键输入
	if !g.animating {  // 只有在没有动画时才处理输入
		if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
			if g.move(DirectionUp) {
				// 移动后自动保存游戏状态，但不显示提醒
				g.saveGame(false)
			}
		} else if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
			if g.move(DirectionRight) {
				g.saveGame(false)
			}
		} else if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
			if g.move(DirectionDown) {
				g.saveGame(false)
			}
		} else if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
			if g.move(DirectionLeft) {
				g.saveGame(false)
			}
		} else if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			// 重置游戏
			g.resetGame()
			g.showMessage("游戏已重置", 60)
		} else if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			// 如果已经赢了，继续游戏
			if g.win && g.showWin {
				g.showWin = false
				g.showMessage("继续游戏", 60)
				g.saveGame(false)
			}
		} else if inpututil.IsKeyJustPressed(ebiten.KeyS) {
			// 手动保存游戏，显示提醒
			g.saveGame(true)
		} else if inpututil.IsKeyJustPressed(ebiten.KeyL) {
			// 手动加载游戏
			g.loadGame()
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

	// 计算分数面板位置，使两侧边距相等
	panelWidth := 100
	leftPanelX := (screenWidth / 2 - panelWidth) / 2 - 45
	rightPanelX := screenWidth / 2 + (screenWidth / 2 - panelWidth) / 2 + 45
	
	// 绘制分数
	drawScorePanel(screen, "分数", g.score, leftPanelX, 90)
	drawScorePanel(screen, "最高分", g.bestScore, rightPanelX, 90)

	// 绘制游戏说明
	instructionText := "R键重置 | S键保存 | L键加载"
	// 计算文本宽度以居中显示
	bounds, _ := font.BoundString(scoreFont, instructionText)
	textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
	
	// 绘制半透明背景确保文字清晰可见
	// ebitenutil.DrawRect(screen, float64(screenWidth/2-textWidth/2-10), 130, float64(textWidth+20), 30, color.RGBA{187, 173, 160, 200})
	text.Draw(screen, instructionText, scoreFont, screenWidth/2-textWidth/2, 150, textColor)

	// 绘制游戏棋盘(只绘制背景和空格)
	drawBoard(screen, g.board)
	
	// 棋盘位置
	boardX := (screenWidth - (tileSize*boardSize + tileMargin*(boardSize-1))) / 2
	boardY := 180
	
	// 如果正在动画中，绘制动画方块
	if g.animating {
		// 先绘制所有非动画方块
		for i := 0; i < boardSize; i++ {
			for j := 0; j < boardSize; j++ {
				// 检查是否是动画目标位置
				isTarget := false
				for _, anim := range g.animations {
					if anim.toX == j && anim.toY == i {
						isTarget = true
						break
					}
				}
				
				// 如果不是动画目标位置，并且当前有方块，则绘制静态方块
				if !isTarget && g.board[i][j] > 0 {
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
		
		// 然后绘制动画中的方块
		for _, anim := range g.animations {
			progress := easeOutQuad(g.animationProgress)
			
			// 计算动画位置
			var currentX, currentY float64
			
			// 移动动画
			currentX = float64(boardX) + float64(anim.fromX)*(tileSize+tileMargin) + 
					   float64(anim.toX-anim.fromX)*(tileSize+tileMargin)*progress
			currentY = float64(boardY) + float64(anim.fromY)*(tileSize+tileMargin) + 
					   float64(anim.toY-anim.fromY)*(tileSize+tileMargin)*progress
			
			// 获取方块颜色
			var tileColor color.RGBA
			if val, ok := tileColors[anim.value]; ok {
				tileColor = val
			} else {
				tileColor = tileColors[2048]
			}
			
			// 如果是合并动画，添加缩放和透明度效果
			var scale, alpha float64
			if anim.animType == AnimationMerge && progress > 0.5 {
				// 在移动完成后半段添加合并效果
				// 先稍微放大
				p := (progress - 0.5) * 2 // 将0.5-1.0映射到0-1.0
				scale = 1.0 + 0.3*sinWave(p) // 使用正弦波实现缩放效果
				
				// 计算透明度变化
				alpha = sinWave(p)
			} else {
				scale = 1.0
				alpha = 1.0
			}
			
			// 计算缩放后的尺寸和位置
			scaledSize := float64(tileSize) * scale
			offsetX := (scaledSize - float64(tileSize)) / 2
			offsetY := (scaledSize - float64(tileSize)) / 2
			
			// 调整颜色透明度
			if anim.animType == AnimationMerge && progress > 0.5 {
				tileColor.A = uint8(255 * alpha)
			}
			
			// 绘制方块
			ebitenutil.DrawRect(screen, currentX-offsetX, currentY-offsetY, scaledSize, scaledSize, tileColor)
			
			// 如果是合并动画且在后半段，不绘制数字（会在目标格子绘制）
			if !(anim.animType == AnimationMerge && progress > 0.85) {
				// 绘制数字
				numStr := fmt.Sprintf("%d", anim.value)
				var tFace font.Face
				
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
				
				textX := int(currentX) + int(scaledSize-float64(textWidth))/2
				textY := int(currentY) + int(scaledSize+float64(textHeight))/2
				
				// 选择文本颜色
				textCol := textColor
				if anim.value > 4 {
					textCol = textColorLight
				}
				
				// 调整文本透明度
				if anim.animType == AnimationMerge && progress > 0.5 {
					textCol = color.RGBA{textCol.R, textCol.G, textCol.B, uint8(255 * alpha)}
				}
				
				// 绘制数字
				text.Draw(screen, numStr, tFace, textX, textY, textCol)
			}
			
			// 如果是合并动画且在靠近结束阶段，绘制目标值的数字
			if anim.animType == AnimationMerge && progress > 0.85 {
				targetX := boardX + anim.toX*(tileSize+tileMargin)
				targetY := boardY + anim.toY*(tileSize+tileMargin)
				targetValue := anim.value * 2
				
				// 获取目标方块颜色和源方块颜色
				var targetColor, sourceColor color.RGBA
				if val, ok := tileColors[targetValue]; ok {
					targetColor = val
				} else {
					targetColor = tileColors[2048]
				}
				
				if val, ok := tileColors[anim.value]; ok {
					sourceColor = val
				} else {
					sourceColor = tileColors[2048]
				}
				
				// 计算从0到1的渐变进度
				fadeInProgress := (progress - 0.85) / 0.15
				
				// 颜色过渡效果
				currentColor := lerpColor(sourceColor, targetColor, fadeInProgress)
				
				// 计算缩放效果 - 使用弹性函数
				targetScale := 1.0 + 0.3*elasticOut(fadeInProgress)
				targetSize := float64(tileSize) * targetScale
				targetOffsetX := (targetSize - float64(tileSize)) / 2
				targetOffsetY := (targetSize - float64(tileSize)) / 2
				
				// 绘制目标方块
				ebitenutil.DrawRect(screen, float64(targetX)-targetOffsetX, float64(targetY)-targetOffsetY, 
									targetSize, targetSize, currentColor)
				
				// 绘制闪光效果
				if fadeInProgress > 0.3 && fadeInProgress < 0.7 {
					glowIntensity := 1.0 - math.Abs(fadeInProgress - 0.5) * 5.0 // 0.5时最强
					glowColor := color.RGBA{255, 255, 255, uint8(100 * glowIntensity)}
					glowSize := targetSize + 10*glowIntensity
					ebitenutil.DrawRect(screen, float64(targetX)-(glowSize-float64(tileSize))/2, 
										float64(targetY)-(glowSize-float64(tileSize))/2, 
										glowSize, glowSize, glowColor)
				}
				
				// 绘制目标数字
				numStr := fmt.Sprintf("%d", targetValue)
				var tFace font.Face
				
				if targetValue < 100 {
					tFace = boldFont
				} else if targetValue < 1000 {
					tFace = boldFont
				} else {
					tFace = boldFont
				}
				
				// 计算文本位置
				bounds, _ := font.BoundString(tFace, numStr)
				textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
				textHeight := (bounds.Max.Y - bounds.Min.Y).Ceil()
				
				textX := targetX + int(targetSize-float64(textWidth))/2
				textY := targetY + int(targetSize+float64(textHeight))/2
				
				// 选择文本颜色
				textCol := textColor
				if targetValue > 4 {
					textCol = textColorLight
				}
				
				// 数字动画效果
				textScale := 1.0 + 0.2*(1.0-fadeInProgress)
				
				// 绘制数字
				op := &ebiten.DrawImageOptions{}
				textImg := ebiten.NewImage(textWidth+10, textHeight+10)
				text.Draw(textImg, numStr, tFace, 5, textHeight, textCol)
				
				op.GeoM.Translate(-float64(textWidth+10)/2, -float64(textHeight+10)/2)
				op.GeoM.Scale(textScale, textScale)
				op.GeoM.Translate(float64(textX+textWidth/2), float64(textY-textHeight/2))
				
				screen.DrawImage(textImg, op)
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
	panelWidth := 100  // 调整面板宽度，使左右间距一致
	panelHeight := 60
	
	// 绘制背景
	ebitenutil.DrawRect(screen, float64(x), float64(y), float64(panelWidth), float64(panelHeight), boardColor)
	
	// 计算标题文本宽度居中显示
	titleBounds, _ := font.BoundString(scoreFont, title)
	titleWidth := (titleBounds.Max.X - titleBounds.Min.X).Ceil()
	titleX := x + (panelWidth - titleWidth) / 2
	
	// 绘制标题
	text.Draw(screen, title, scoreFont, titleX, y+20, textColorLight)
	
	// 计算分数文本宽度居中显示
	scoreText := fmt.Sprintf("%d", score)
	scoreBounds, _ := font.BoundString(boldFont, scoreText)
	scoreWidth := (scoreBounds.Max.X - scoreBounds.Min.X).Ceil()
	scoreX := x + (panelWidth - scoreWidth) / 2
	
	// 绘制分数
	text.Draw(screen, scoreText, boldFont, scoreX, y+45, textColorLight)
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
	
	// 程序正常退出时保存游戏并显示提醒
	game.saveGame(true)
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

// 正弦波函数 - 用于制作平滑的缩放和淡入淡出效果
func sinWave(t float64) float64 {
	return math.Sin(t * math.Pi / 2)
}

// 颜色渐变函数 - 在两个颜色之间平滑过渡
func lerpColor(c1, c2 color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(c1.R) + t*(float64(c2.R)-float64(c1.R))),
		G: uint8(float64(c1.G) + t*(float64(c2.G)-float64(c1.G))),
		B: uint8(float64(c1.B) + t*(float64(c2.B)-float64(c1.B))),
		A: uint8(float64(c1.A) + t*(float64(c2.A)-float64(c1.A))),
	}
}

// 弹性函数 - 制作有弹性的效果
func elasticOut(t float64) float64 {
	if t == 0 || t == 1 {
		return t
	}
	p := 0.3 // 弹性系数
	s := p / 4.0
	return math.Pow(2, -10*t) * math.Sin((t-s)*(2*math.Pi)/p) + 1
} 